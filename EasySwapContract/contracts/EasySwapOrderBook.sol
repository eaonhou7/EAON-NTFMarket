// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import {
    Initializable
} from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import {
    ContextUpgradeable
} from "@openzeppelin/contracts-upgradeable/utils/ContextUpgradeable.sol";
import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {
    ReentrancyGuardUpgradeable
} from "@openzeppelin/contracts-upgradeable/utils/ReentrancyGuardUpgradeable.sol";
import {
    PausableUpgradeable
} from "@openzeppelin/contracts-upgradeable/utils/PausableUpgradeable.sol";

import {
    LibTransferSafeUpgradeable,
    IERC721
} from "./libraries/LibTransferSafeUpgradeable.sol";
import {Price} from "./libraries/RedBlackTreeLibrary.sol";
import {LibOrder, OrderKey} from "./libraries/LibOrder.sol";
import {LibPayInfo} from "./libraries/LibPayInfo.sol";

import {IEasySwapOrderBook} from "./interface/IEasySwapOrderBook.sol";
import {IEasySwapVault} from "./interface/IEasySwapVault.sol";

import {OrderStorage} from "./OrderStorage.sol";
import {OrderValidator} from "./OrderValidator.sol";
import {ProtocolManager} from "./ProtocolManager.sol";

// EasySwap 核心订单簿合约
// 继承了 OpenZeppelin 的多个升级模块以及内部功能模块
contract EasySwapOrderBook is
    IEasySwapOrderBook,
    Initializable,
    ContextUpgradeable,
    OwnableUpgradeable,
    ReentrancyGuardUpgradeable,
    PausableUpgradeable,
    OrderStorage,
    ProtocolManager,
    OrderValidator
{
    using LibTransferSafeUpgradeable for address;
    using LibTransferSafeUpgradeable for IERC721;

    // 挂单事件：记录订单创建的关键信息
    // @param orderKey: 订单唯一哈希
    // @param side: 买卖方向 (Bid/List)
    // @param saleKind: 销售类型 (定价/拍卖等)
    // @param maker: 挂单人地址
    // @param nft: NFT 资产信息
    // @param price: 价格
    // @param expiry: 过期时间
    // @param salt: 随机盐值
    event LogMake(
        OrderKey orderKey,
        LibOrder.Side indexed side,
        LibOrder.SaleKind indexed saleKind,
        address indexed maker,
        LibOrder.Asset nft,
        Price price,
        uint64 expiry,
        uint64 salt
    );

    // 取消订单事件
    // @param orderKey: 被取消的订单哈希
    // @param maker: 订单发起人
    event LogCancel(OrderKey indexed orderKey, address indexed maker);

    // 订单匹配（成交）事件
    // @param makeOrderKey: 挂单 (Maker) 的哈希
    // @param takeOrderKey: 吃单 (Taker) 的哈希
    // @param makeOrder: 挂单详情
    // @param takeOrder: 吃单详情
    // @param fillPrice: 最终成交价格
    event LogMatch(
        OrderKey indexed makeOrderKey,
        OrderKey indexed takeOrderKey,
        LibOrder.Order makeOrder,
        LibOrder.Order takeOrder,
        uint128 fillPrice
    );

    // 提取协议收入或多余资金的事件
    event LogWithdrawETH(address recipient, uint256 amount);

    // 批量匹配中单个匹配失败的错误记录 (不影响其他匹配)
    event BatchMatchInnerError(uint256 offset, bytes msg);

    // 订单创建或取消过程中因校验失败被跳过的事件 (软失败)
    event LogSkipOrder(OrderKey orderKey, uint64 salt);

    // 修饰符: 仅允许 delegatecall 调用
    // 用于批量匹配逻辑中，确保某些函数只能通过 delegatecall 上下文执行
    modifier onlyDelegateCall() {
        _checkDelegateCall();
        _;
    }

    /// @custom:oz-upgrades-unsafe-allow state-variable-immutable state-variable-assignment
    // 记录合约部署时的自身地址，用于检测是否在 delegatecall 环境中
    // immutable 变量存储在代码段中，而非 storage
    address private immutable self = address(this);

    // 资金库合约地址，用于托管 NFT 和 ETH
    address private _vault;

    /**
     * @notice 初始化合约 (代理模式)
     * @param newProtocolShare 初始协议费率 (万分位)
     * @param newVault 资金库合约地址
     * @param EIP712Name EIP712 签名域名称
     * @param EIP712Version EIP712 签名域版本
     */
    function initialize(
        uint128 newProtocolShare,
        address newVault,
        string memory EIP712Name,
        string memory EIP712Version
    ) public initializer {
        __EasySwapOrderBook_init(
            newProtocolShare,
            newVault,
            EIP712Name,
            EIP712Version
        );
    }

    // 内部初始化函数
    function __EasySwapOrderBook_init(
        uint128 newProtocolShare,
        address newVault,
        string memory EIP712Name,
        string memory EIP712Version
    ) internal onlyInitializing {
        __EasySwapOrderBook_init_unchained(
            newProtocolShare,
            newVault,
            EIP712Name,
            EIP712Version
        );
    }

    // 执行各模块的初始化逻辑
    function __EasySwapOrderBook_init_unchained(
        uint128 newProtocolShare,
        address newVault,
        string memory EIP712Name,
        string memory EIP712Version
    ) internal onlyInitializing {
        // 初始化上下文、权限、防重入和暂停模块
        __Context_init();
        __Ownable_init(_msgSender());
        __ReentrancyGuard_init();
        __Pausable_init();

        // 初始化业务模块
        __OrderStorage_init();
        __ProtocolManager_init(newProtocolShare);
        __OrderValidator_init(EIP712Name, EIP712Version);

        // 设置 Vault 地址
        setVault(newVault);
    }

    /**
     * @notice 创建多个订单并转移相关资产
     * @dev 如果是卖单 (List)，将 NFT 转移到 Vault
     * @dev 如果是买单 (Bid)，需附带 ETH (msg.value)，并将 ETH 存入 Vault
     * @param newOrders 订单列表
     * @return newOrderKeys 返回成功创建的订单 ID 列表
     */
    function makeOrders(
        LibOrder.Order[] calldata newOrders
    )
        external
        payable
        override
        whenNotPaused
        nonReentrant
        returns (OrderKey[] memory newOrderKeys)
    {
        uint256 orderAmount = newOrders.length;
        newOrderKeys = new OrderKey[](orderAmount);

        uint128 ETHAmount; // 累计需要锁定的 ETH 总额 (用于校验 msg.value)
        for (uint256 i = 0; i < orderAmount; ++i) {
            uint128 buyPrice; // 单个买单所需金额
            if (newOrders[i].side == LibOrder.Side.Bid) {
                // 买单金额 = 单价 * 数量
                buyPrice =
                    Price.unwrap(newOrders[i].price) *
                    newOrders[i].nft.amount;
            }

            // 尝试创建订单，返回生成的 Key (如果失败则返回 Sentinel)
            OrderKey newOrderKey = _makeOrderTry(newOrders[i], buyPrice);
            newOrderKeys[i] = newOrderKey;

            // 如果创建成功，累计所需的 ETH 金额
            if (
                OrderKey.unwrap(newOrderKey) !=
                OrderKey.unwrap(LibOrder.ORDERKEY_SENTINEL)
            ) {
                ETHAmount += buyPrice;
            }
        }

        // 如果用户发送的 ETH 多于所需金额，退还多余部分
        if (msg.value > ETHAmount) {
            _msgSender().safeTransferETH(msg.value - ETHAmount);
        }
    }

    /**
     * @dev 批量取消订单
     * @param orderKeys 要取消的订单 Hash 列表
     * @return successes 取消结果列表
     */
    function cancelOrders(
        OrderKey[] calldata orderKeys
    )
        external
        override
        whenNotPaused
        nonReentrant
        returns (bool[] memory successes)
    {
        successes = new bool[](orderKeys.length);

        for (uint256 i = 0; i < orderKeys.length; ++i) {
            // 调用尝试取消逻辑
            bool success = _cancelOrderTry(orderKeys[i]);
            successes[i] = success;
        }
    }

    /**
     * @notice 批量编辑订单 (取消旧订单，创建新订单)
     * @dev 用于快速改价，原子性操作
     * @param editDetails 编辑详情列表 (包含旧 OrderKey 和新 Order 结构)
     * @return newOrderKeys 新生成的订单 Key 列表
     */
    function editOrders(
        LibOrder.EditDetail[] calldata editDetails
    )
        external
        payable
        override
        whenNotPaused
        nonReentrant
        returns (OrderKey[] memory newOrderKeys)
    {
        newOrderKeys = new OrderKey[](editDetails.length);

        uint256 bidETHAmount; // 累计需要补足的 ETH
        for (uint256 i = 0; i < editDetails.length; ++i) {
            // 尝试编辑订单，返回新 Key 和可能需要补足的差价
            (OrderKey newOrderKey, uint256 bidPrice) = _editOrderTry(
                editDetails[i].oldOrderKey,
                editDetails[i].newOrder
            );
            bidETHAmount += bidPrice;
            newOrderKeys[i] = newOrderKey;
        }

        // 如果用户发送的 ETH 多于所需补足金额，退还多余部分
        if (msg.value > bidETHAmount) {
            _msgSender().safeTransferETH(msg.value - bidETHAmount);
        }
    }

    // 匹配单个订单
    // @param sellOrder: 卖单信息
    // @param buyOrder: 买单信息
    function matchOrder(
        LibOrder.Order calldata sellOrder,
        LibOrder.Order calldata buyOrder
    ) external payable override whenNotPaused nonReentrant {
        // 执行匹配逻辑，返回实际消耗的 value (如果是买家主动吃单且未存款的情况)
        uint256 costValue = _matchOrder(sellOrder, buyOrder, msg.value);
        // 如果 msg.value 有剩，退还给调用者
        if (msg.value > costValue) {
            _msgSender().safeTransferETH(msg.value - costValue);
        }
    }

    /**
     * @dev 原子化批量匹配订单
     * @dev 使用 delegatecall 的目的是为了隔离每个匹配操作的上下文，
     *      如果单个匹配失败 (revert)，通过 try/catch (在底层是 call 返回 false) 捕获，
     *      仅回滚该子交易，不影响批量中的其他匹配。
     * @param matchDetails 匹配对列表
     * @return successes 匹配结果列表
     */
    /// @custom:oz-upgrades-unsafe-allow delegatecall
    function matchOrders(
        LibOrder.MatchDetail[] calldata matchDetails
    )
        external
        payable
        override
        whenNotPaused
        nonReentrant
        returns (bool[] memory successes)
    {
        successes = new bool[](matchDetails.length);

        uint128 buyETHAmount; // 累计买家 (如果是调用者) 消耗的 ETH

        for (uint256 i = 0; i < matchDetails.length; ++i) {
            LibOrder.MatchDetail calldata matchDetail = matchDetails[i];

            // 使用 delegatecall 调用当前合约的 matchOrderWithoutPayback 函数
            // 这样可以在保留当前上下文 (msg.sender) 的同时，捕获 revert
            (bool success, bytes memory data) = address(this).delegatecall(
                abi.encodeWithSignature(
                    "matchOrderWithoutPayback((uint8,uint8,address,(uint256,address,uint96),uint128,uint64,uint64),(uint8,uint8,address,(uint256,address,uint96),uint128,uint64,uint64),uint256)",
                    matchDetail.sellOrder,
                    matchDetail.buyOrder,
                    msg.value - buyETHAmount // 传入剩余可用的 msg.value
                )
            );

            if (success) {
                successes[i] = success;
                if (matchDetail.buyOrder.maker == _msgSender()) {
                    // 如果调用者是买方 (Take List)，记录其花费的 ETH
                    // matchOrderWithoutPayback 返回实际消耗的 costValue
                    uint128 buyPrice;
                    buyPrice = abi.decode(data, (uint128));
                    buyETHAmount += buyPrice;
                }
            } else {
                // 如果匹配失败，触发事件记录错误信息
                emit BatchMatchInnerError(i, data);
            }
        }

        // 批量结束后，统一退还多余的 ETH
        if (msg.value > buyETHAmount) {
            _msgSender().safeTransferETH(msg.value - buyETHAmount);
        }
    }

    // 用于 delegatecall 的匹配函数，不包含退款逻辑 (避免在子调用中重复退款导致错误)
    // 仅供 matchOrders 内部调用
    function matchOrderWithoutPayback(
        LibOrder.Order calldata sellOrder,
        LibOrder.Order calldata buyOrder,
        uint256 msgValue
    )
        external
        payable
        whenNotPaused
        onlyDelegateCall // 确保只能通过 delegatecall 调用
        returns (uint128 costValue)
    {
        costValue = _matchOrder(sellOrder, buyOrder, msgValue);
    }

    // 内部尝试创建订单逻辑
    function _makeOrderTry(
        LibOrder.Order calldata order,
        uint128 ETHAmount
    ) internal returns (OrderKey newOrderKey) {
        // 校验基础条件:
        // 1. 调用者必须是 Maker
        // 2. 价格、Salt 不能为 0
        // 3. 未过期
        // 4. 订单哈希未被使用 (防止重放或重复挂单)
        if (
            order.maker == _msgSender() &&
            Price.unwrap(order.price) != 0 &&
            order.salt != 0 &&
            (order.expiry > block.timestamp || order.expiry == 0) &&
            filledAmount[LibOrder.hash(order)] == 0
        ) {
            newOrderKey = LibOrder.hash(order);

            // 将资产存入 Vault
            if (order.side == LibOrder.Side.List) {
                // 卖单: 必须是 1 个 NFT (ERC721 限制)
                if (order.nft.amount != 1) {
                    return LibOrder.ORDERKEY_SENTINEL;
                }
                // 转移 NFT 到 Vault
                IEasySwapVault(_vault).depositNFT(
                    newOrderKey,
                    order.maker,
                    order.nft.collection,
                    order.nft.tokenId
                );
            } else if (order.side == LibOrder.Side.Bid) {
                // 买单: 数量不能为 0
                if (order.nft.amount == 0) {
                    return LibOrder.ORDERKEY_SENTINEL;
                }
                // 存款 ETH 到 Vault
                IEasySwapVault(_vault).depositETH{value: uint256(ETHAmount)}(
                    newOrderKey,
                    ETHAmount
                );
            }

            // 添加到 OrderStorage 存储
            _addOrder(order);

            // 触发挂单事件
            emit LogMake(
                newOrderKey,
                order.side,
                order.saleKind,
                order.maker,
                order.nft,
                order.price,
                order.expiry,
                order.salt
            );
        } else {
            // 校验失败，跳过该订单 (软失败)
            emit LogSkipOrder(LibOrder.hash(order), order.salt);
        }
    }

    // 内部尝试取消订单逻辑
    function _cancelOrderTry(
        OrderKey orderKey
    ) internal returns (bool success) {
        // 从 storage 获取订单信息
        LibOrder.Order memory order = orders[orderKey].order;

        // 校验:
        // 1. 调用者必须是 Maker
        // 2. 订单未完全成交 (filledAmount < amount)
        if (
            order.maker == _msgSender() &&
            filledAmount[orderKey] < order.nft.amount
        ) {
            OrderKey orderHash = LibOrder.hash(order);

            // 从链表和树中移除订单数据
            _removeOrder(order);

            // 从 Vault 撤回资产
            if (order.side == LibOrder.Side.List) {
                // 卖单: 取回 NFT
                IEasySwapVault(_vault).withdrawNFT(
                    orderHash,
                    order.maker,
                    order.nft.collection,
                    order.nft.tokenId
                );
            } else if (order.side == LibOrder.Side.Bid) {
                // 买单: 取回剩余的 ETH
                // 计算剩余未成交金额 = 单价 * (总量 - 已成交量)
                uint256 availNFTAmount = order.nft.amount -
                    filledAmount[orderKey];
                IEasySwapVault(_vault).withdrawETH(
                    orderHash,
                    Price.unwrap(order.price) * availNFTAmount,
                    order.maker
                );
            }
            // 标记订单状态为已取消 (filledAmount = MAX)
            _cancelOrder(orderKey);
            success = true;
            emit LogCancel(orderKey, order.maker);
        } else {
            // 校验失败，跳过
            emit LogSkipOrder(orderKey, order.salt);
        }
    }

    // 内部尝试编辑订单逻辑
    function _editOrderTry(
        OrderKey oldOrderKey,
        LibOrder.Order calldata newOrder
    ) internal returns (OrderKey newOrderKey, uint256 deltaBidPrice) {
        LibOrder.Order memory oldOrder = orders[oldOrderKey].order;

        // 检查新旧订单的一致性 (除了价格，其他核心参数必须一致)
        if (
            (oldOrder.saleKind != newOrder.saleKind) ||
            (oldOrder.side != newOrder.side) ||
            (oldOrder.maker != newOrder.maker) ||
            (oldOrder.nft.collection != newOrder.nft.collection) ||
            (oldOrder.nft.tokenId != newOrder.nft.tokenId) ||
            filledAmount[oldOrderKey] >= oldOrder.nft.amount // 旧订单不能已成交或取消
        ) {
            emit LogSkipOrder(oldOrderKey, oldOrder.salt);
            return (LibOrder.ORDERKEY_SENTINEL, 0);
        }

        // 检查新订单基本参数有效性
        if (
            newOrder.maker != _msgSender() ||
            newOrder.salt == 0 ||
            (newOrder.expiry < block.timestamp && newOrder.expiry != 0) ||
            filledAmount[LibOrder.hash(newOrder)] != 0 // 新订单 Hash 不能已存在
        ) {
            emit LogSkipOrder(oldOrderKey, newOrder.salt);
            return (LibOrder.ORDERKEY_SENTINEL, 0);
        }

        // 取消旧订单
        uint256 oldFilledAmount = filledAmount[oldOrderKey];
        _removeOrder(oldOrder); // 从存储移除
        _cancelOrder(oldOrderKey); // 标记取消
        emit LogCancel(oldOrderKey, oldOrder.maker);

        // 添加新订单
        newOrderKey = _addOrder(newOrder);

        // 调整 Vault 中的资产
        if (oldOrder.side == LibOrder.Side.List) {
            // 卖单：NFT 在 Vault 中，只需把记录从旧 Key 转移到新 Key
            IEasySwapVault(_vault).editNFT(oldOrderKey, newOrderKey);
        } else if (oldOrder.side == LibOrder.Side.Bid) {
            // 买单：需要重新计算并多退少补 ETH
            uint256 oldRemainingPrice = Price.unwrap(oldOrder.price) *
                (oldOrder.nft.amount - oldFilledAmount);
            uint256 newRemainingPrice = Price.unwrap(newOrder.price) *
                newOrder.nft.amount;

            if (newRemainingPrice > oldRemainingPrice) {
                // 涨价：用户需要补足差额 (deltaBidPrice)
                deltaBidPrice = newRemainingPrice - oldRemainingPrice;
                // 注意：这里调用 Vault 的 editETH，附带 msg.value 的一部分 (由外部循环处理传入)
                IEasySwapVault(_vault).editETH{value: uint256(deltaBidPrice)}(
                    oldOrderKey,
                    newOrderKey,
                    oldRemainingPrice,
                    newRemainingPrice,
                    oldOrder.maker
                );
            } else {
                // 降价：Vault 退还差额给 Maker
                IEasySwapVault(_vault).editETH(
                    oldOrderKey,
                    newOrderKey,
                    oldRemainingPrice,
                    newRemainingPrice,
                    oldOrder.maker
                );
            }
        }

        emit LogMake(
            newOrderKey,
            newOrder.side,
            newOrder.saleKind,
            newOrder.maker,
            newOrder.nft,
            newOrder.price,
            newOrder.expiry,
            newOrder.salt
        );
    }

    // 核心匹配逻辑
    function _matchOrder(
        LibOrder.Order calldata sellOrder,
        LibOrder.Order calldata buyOrder,
        uint256 msgValue
    ) internal returns (uint128 costValue) {
        OrderKey sellOrderKey = LibOrder.hash(sellOrder);
        OrderKey buyOrderKey = LibOrder.hash(buyOrder);

        // 1. 预检查撮合条件 (方向、资产匹配、未关闭等)
        _isMatchAvailable(sellOrder, buyOrder, sellOrderKey, buyOrderKey);

        if (_msgSender() == sellOrder.maker) {
            // --- 情况 1: 卖家主动吃单 (Take Bid) ---
            // 卖家是 Taker，买单 (Make Order) 已在 OrderBook 中

            // 卖单作为 Taker 不能附带 ETH
            require(msgValue == 0, "HD: value > 0");

            // 检查卖单是否也已经在 OrderBook 中 (可能之前挂了单现在自己吃，或者部分成交)
            bool isSellExist = orders[sellOrderKey].order.maker != address(0);
            _validateOrder(sellOrder, isSellExist);
            // 验证链上已有的买单信息
            _validateOrder(orders[buyOrderKey].order, false);

            uint128 fillPrice = Price.unwrap(buyOrder.price); // 以挂单(Maker) 的价格成交

            if (isSellExist) {
                // 如果卖单也在链上，需先移除卖单记录，避免状态冲突
                _removeOrder(sellOrder);
                _updateFilledAmount(sellOrder.nft.amount, sellOrderKey); // 标记卖单已完全成交
            }
            // 更新买单填充量 (+1)
            _updateFilledAmount(filledAmount[buyOrderKey] + 1, buyOrderKey);

            emit LogMatch(
                sellOrderKey,
                buyOrderKey,
                sellOrder,
                buyOrder,
                fillPrice
            );

            // --- 资金结算 ---
            // 1. Vault 从 Buy Order 的余额中提取 ETH 到当前 OrderBook 合约
            IEasySwapVault(_vault).withdrawETH(
                buyOrderKey,
                fillPrice,
                address(this)
            );

            // 2. 扣除协议费，将剩余 ETH 转给卖家 (Taker)
            uint128 protocolFee = _shareToAmount(fillPrice, protocolShare);
            sellOrder.maker.safeTransferETH(fillPrice - protocolFee);

            // 3. NFT 转移 (Vault -> Buyer 或 Maker -> Buyer)
            if (isSellExist) {
                // 卖单在链上，说明 NFT 已经在 Vault 中
                IEasySwapVault(_vault).withdrawNFT(
                    sellOrderKey,
                    buyOrder.maker,
                    sellOrder.nft.collection,
                    sellOrder.nft.tokenId
                );
            } else {
                // 卖单不在链上，NFT 在卖家手中，直接从 Maker 转给 Buyer
                // (注意：这里调用 Vault 的 transferERC721 只是为了统一接口或利用 Vault 的权限/逻辑?)
                // 实际上 Vault 应该有权限操作 NFT? 或者这里仅仅是辅助
                // 根据代码，Vault 的 transferERC721 是调用 collection.safeTransferFrom(from, to)
                // 所以卖家必须 approve Vault 合约地址，或者 OrderBook (如果是 OrderBook 调用的)
                // 这里的 _vault.transferERC721(from, to) 需要 Vault 有 operator 权限
                IEasySwapVault(_vault).transferERC721(
                    sellOrder.maker,
                    buyOrder.maker,
                    sellOrder.nft
                );
            }
        } else if (_msgSender() == buyOrder.maker) {
            // --- 情况 2: 买家主动吃单 (Take List) ---
            // 买家是 Taker，卖单 (Make Order) 已在 OrderBook 中

            bool isBuyExist = orders[buyOrderKey].order.maker != address(0);

            // 验证卖单 (Maker)
            _validateOrder(orders[sellOrderKey].order, false);
            // 验证买单 (Taker)
            _validateOrder(buyOrder, isBuyExist);

            uint128 buyPrice = Price.unwrap(buyOrder.price);
            uint128 fillPrice = Price.unwrap(sellOrder.price); // 以卖单(Maker)的价格成交

            if (!isBuyExist) {
                // 买单不在链上，买家直接支付 ETH (使用 msg.value)
                require(msgValue >= fillPrice, "HD: value < fill price");
            } else {
                // 买单在链上 (之前挂了 Buy Order)，Vault 中有 ETH
                require(buyPrice >= fillPrice, "HD: buy price < fill price");
                // 从 Vault 提取 ETH 支付本次成交
                IEasySwapVault(_vault).withdrawETH(
                    buyOrderKey,
                    buyPrice,
                    address(this)
                );
                // 移除买单记录 (已成交)
                _removeOrder(buyOrder);
                _updateFilledAmount(filledAmount[buyOrderKey] + 1, buyOrderKey);
            }
            // 更新卖单状态 (NFT 卖单通常只有 1 个，成交即满)
            _updateFilledAmount(sellOrder.nft.amount, sellOrderKey);

            emit LogMatch(
                buyOrderKey,
                sellOrderKey,
                buyOrder,
                sellOrder,
                fillPrice
            );

            // --- 资金结算 ---
            uint128 protocolFee = _shareToAmount(fillPrice, protocolShare);
            // 转账给卖家
            sellOrder.maker.safeTransferETH(fillPrice - protocolFee);

            // 如果买家支付/锁定的金额高于成交价 (例如之前挂的高价 Bid 被低价 List 吃掉? 不，这里是 Take List)
            // 场景：Bid Price > Sell Price，按 Sell Price 成交，退还差价
            if (buyPrice > fillPrice) {
                buyOrder.maker.safeTransferETH(buyPrice - fillPrice);
            }

            // NFT 从 Vault 转移给买家
            IEasySwapVault(_vault).withdrawNFT(
                sellOrderKey,
                buyOrder.maker,
                sellOrder.nft.collection,
                sellOrder.nft.tokenId
            );

            // 返回本次交易买家实际花费的 ETH (用于批量退款计算)
            // 如果买单已存在 (资金在 Vault)，则 msg.value 消耗为 0
            // 如果买单不存在 (资金由 msg.value 提供)，消耗为 buyPrice (其实是 fillPrice? 但代码写的 buyPrice)
            // 修正理解：如果买家直接支付，消耗的是 fillPrice，但代码逻辑似乎假设 buyPrice
            // 实际上如果 !isBuyExist, msgValue >= fillPrice，退还差价在 matchOrder 外层处理
            // 这里返回 costValue 给 matchOrder
            costValue = isBuyExist ? 0 : buyPrice;
        } else {
            revert("HD: sender invalid");
        }
    }

    // 检查撮合的必要条件
    function _isMatchAvailable(
        LibOrder.Order memory sellOrder,
        LibOrder.Order memory buyOrder,
        OrderKey sellOrderKey,
        OrderKey buyOrderKey
    ) internal view {
        // 同一个订单哈希不能自我撮合
        require(
            OrderKey.unwrap(sellOrderKey) != OrderKey.unwrap(buyOrderKey),
            "HD: same order"
        );
        // 必须是一买一卖
        require(
            sellOrder.side == LibOrder.Side.List &&
                buyOrder.side == LibOrder.Side.Bid,
            "HD: side mismatch"
        );
        // 卖单必须是 FixedPriceForItem (目前仅支持按固定价格出售单品)
        require(
            sellOrder.saleKind == LibOrder.SaleKind.FixedPriceForItem,
            "HD: kind mismatch"
        );
        // 买卖双方不能是同一人
        require(sellOrder.maker != buyOrder.maker, "HD: same maker");

        // 资产必须匹配
        // 如果买单是 Collection Offer (求购任意单品)，则只校验 Collection
        // 如果买单是 Item Offer (求购特定单品)，则还需校验 TokenID
        require(
            // check if the asset is the same
            buyOrder.saleKind == LibOrder.SaleKind.FixedPriceForCollection ||
                (sellOrder.nft.collection == buyOrder.nft.collection &&
                    sellOrder.nft.tokenId == buyOrder.nft.tokenId),
            "HD: asset mismatch"
        );

        // 订单必须未完结 (剩余量 > 已填充量)
        require(
            filledAmount[sellOrderKey] < sellOrder.nft.amount &&
                filledAmount[buyOrderKey] < buyOrder.nft.amount,
            "HD: order closed"
        );
    }

    /**
     * @notice 计算分成金额
     * @param total 总金额
     * @param share 分成比例 (万分位)
     * @return 分成对应的金额
     */
    function _shareToAmount(
        uint128 total,
        uint128 share
    ) internal pure returns (uint128) {
        return (total * share) / LibPayInfo.TOTAL_SHARE;
    }

    // 检查 delegatecall 环境
    function _checkDelegateCall() private view {
        // 如果 address(this) 不等于 immutable 的 self，说明是在 delegatecall 环境中
        require(address(this) != self);
    }

    // 设置 Vault
    function setVault(address newVault) public onlyOwner {
        require(newVault != address(0), "HD: zero address");
        _vault = newVault;
    }

    // 提取协议收入 (ETH)
    function withdrawETH(
        address recipient,
        uint256 amount
    ) external nonReentrant onlyOwner {
        recipient.safeTransferETH(amount);
        emit LogWithdrawETH(recipient, amount);
    }

    // 暂停合约
    function pause() external onlyOwner {
        _pause();
    }

    // 恢复合约
    function unpause() external onlyOwner {
        _unpause();
    }

    // 接收 ETH
    receive() external payable {}

    // 预留存储空间
    uint256[50] private __gap;
}
