// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {
    LibTransferSafeUpgradeable,
    IERC721
} from "./libraries/LibTransferSafeUpgradeable.sol";
import {LibOrder, OrderKey} from "./libraries/LibOrder.sol";

import {IEasySwapVault} from "./interface/IEasySwapVault.sol";

// 定义 EasySwapVault 合约，实现 IEasySwapVault 接口和 OwnableUpgradeable (权限控制)
contract EasySwapVault is IEasySwapVault, OwnableUpgradeable {
    using LibTransferSafeUpgradeable for address;
    using LibTransferSafeUpgradeable for IERC721;

    // 市场合约地址，只有该地址有权限调用 vault 的核心功能
    // 该地址在初始化后通过 setOrderBook 设置
    address public orderBook;

    // 记录每个订单锁定的 ETH 余额
    // Key: OrderKey (订单哈希)
    // Value: 锁定的 ETH 数量 (单位: wei)
    mapping(OrderKey => uint256) public ETHBalance;

    // 记录每个订单锁定的 NFT TokenId
    // Key: OrderKey (订单哈希)
    // Value: NFT 的 Token ID
    // 注意: 这里假设一个订单只锁定一个 TokenID (对于 ERC721)
    mapping(OrderKey => uint256) public NFTBalance;

    // 修饰符: 仅允许 EasySwapOrderBook 合约调用
    modifier onlyEasySwapOrderBook() {
        // 校验调用者必须是 orderBook 地址
        require(msg.sender == orderBook, "HV: only EasySwap OrderBook");
        _;
    }

    // 初始化函数，用于代理合约
    function initialize() public initializer {
        // 初始化 Owner 权限
        __Ownable_init(_msgSender());
    }

    // 设置 OrderBook 合约地址
    // 仅合约 Owner 可以调用
    // @param newOrderBook: 新的 OrderBook 地址
    function setOrderBook(address newOrderBook) public onlyOwner {
        // 校验地址非零
        require(newOrderBook != address(0), "HV: zero address");
        orderBook = newOrderBook;
    }

    // 查询订单的资产余额 (ETH 和 NFT)
    // @param orderKey: 订单哈希
    // @return ETHAmount: 订单关联的 ETH 余额
    // @return tokenId: 订单关联的 NFT ID
    function balanceOf(
        OrderKey orderKey
    ) external view returns (uint256 ETHAmount, uint256 tokenId) {
        ETHAmount = ETHBalance[orderKey];
        tokenId = NFTBalance[orderKey];
    }

    // 存入 ETH (用于 Bid 买单)
    // @param orderKey: 订单哈希
    // @param ETHAmount: 需要存入的 ETH 数量
    function depositETH(
        OrderKey orderKey,
        uint256 ETHAmount
    ) external payable onlyEasySwapOrderBook {
        // 校验: 发送的 ETH 必须大于等于声称的数量
        require(msg.value >= ETHAmount, "HV: not match ETHAmount");
        // 增加该订单的 ETH 余额记录
        ETHBalance[orderKey] += msg.value;
    }

    // 提取 ETH (用于成交、取消或退款)
    // @param orderKey: 订单哈希
    // @param ETHAmount: 提取金额
    // @param to: 接收 ETH 的地址
    function withdrawETH(
        OrderKey orderKey,
        uint256 ETHAmount,
        address to
    ) external onlyEasySwapOrderBook {
        // 扣除订单的 ETH 余额
        // Solidity 0.8+ 会自动检查下溢出
        ETHBalance[orderKey] -= ETHAmount;
        // 安全地将 ETH 转给接收者
        to.safeTransferETH(ETHAmount);
    }

    // 存入 NFT (用于 List 卖单)
    // @param orderKey: 订单哈希
    // @param from: NFT 来源地址 (卖家)
    // @param collection: NFT 合约地址
    // @param tokenId: NFT ID
    function depositNFT(
        OrderKey orderKey,
        address from,
        address collection,
        uint256 tokenId
    ) external onlyEasySwapOrderBook {
        // 使用 safeTransferNFT 将 NFT 从卖家转移到 vault 合约
        // 注意: 卖家必须先 approve vault 或 orderbook (取决于实现细节, 这里是 vault)
        IERC721(collection).safeTransferNFT(from, address(this), tokenId);

        // 记录该订单对应的 NFT TokenId
        NFTBalance[orderKey] = tokenId;
    }

    // 提取 NFT (用于取消或成交)
    // @param orderKey: 订单哈希
    // @param to: 接收 NFT 的地址
    // @param collection: NFT 合约地址
    // @param tokenId: NFT ID
    function withdrawNFT(
        OrderKey orderKey,
        address to,
        address collection,
        uint256 tokenId
    ) external onlyEasySwapOrderBook {
        // 校验: 确保存储的 TokenID 与请求的一致 (防止提取错资产)
        require(NFTBalance[orderKey] == tokenId, "HV: not match tokenId");
        // 删除 NFT 余额记录
        delete NFTBalance[orderKey];

        // 将 NFT 从 Vault 转移给接收者
        IERC721(collection).safeTransferNFT(address(this), to, tokenId);
    }

    // 编辑订单时更新 ETH 余额
    // 当买单价格调整时，需要多退少补
    // @param oldOrderKey: 旧订单哈希
    // @param newOrderKey: 新订单哈希
    // @param oldETHAmount: 旧订单锁定的金额
    // @param newETHAmount: 新订单需要的金额
    // @param to: 如果需要退款，接收退款的地址 (通常是 Maker)
    function editETH(
        OrderKey oldOrderKey,
        OrderKey newOrderKey,
        uint256 oldETHAmount,
        uint256 newETHAmount,
        address to
    ) external payable onlyEasySwapOrderBook {
        // 清除旧订单的 ETH 余额记录
        ETHBalance[oldOrderKey] = 0;

        if (oldETHAmount > newETHAmount) {
            // 情况 1: 旧金额 > 新金额 (降价)，需要退还差价
            // 设置新订单的余额
            ETHBalance[newOrderKey] = newETHAmount;
            // 将差额退还给用户
            to.safeTransferETH(oldETHAmount - newETHAmount);
        } else if (oldETHAmount < newETHAmount) {
            // 情况 2: 旧金额 < 新金额 (涨价)，需要用户补足差价
            // 校验: 用户发送的 ETH 是否足够补差价
            require(
                msg.value >= newETHAmount - oldETHAmount,
                "HV: not match newETHAmount"
            );
            // 设置新订单余额 = 补的钱 + 旧余额
            ETHBalance[newOrderKey] = msg.value + oldETHAmount;
        } else {
            // 情况 3: 金额不变，直接转移记录到新订单
            ETHBalance[newOrderKey] = oldETHAmount;
        }
    }

    // 编辑订单时更新 NFT 记录
    // 对于 NFT 卖单，通常只需要转移记录，不需要实际发生 NFT 转移 (仍在 Vault 中)
    // @param oldOrderKey: 旧订单哈希
    // @param newOrderKey: 新订单哈希
    function editNFT(
        OrderKey oldOrderKey,
        OrderKey newOrderKey
    ) external onlyEasySwapOrderBook {
        // 将旧订单的 NFT 记录转移给新订单
        NFTBalance[newOrderKey] = NFTBalance[oldOrderKey];
        // 删除旧订单记录
        delete NFTBalance[oldOrderKey];
    }

    // 辅助函数：直接转移 ERC721 (不经过 Vault 存储)
    // 用于撮合逻辑中，当资产不在 Vault 中但在卖家钱包时 (例如未托管模式)
    // @param from: 发送方
    // @param to: 接收方
    // @param assets: NFT 资产信息
    function transferERC721(
        address from,
        address to,
        LibOrder.Asset calldata assets
    ) external onlyEasySwapOrderBook {
        // 直接从 from 转移到 to
        IERC721(assets.collection).safeTransferNFT(from, to, assets.tokenId);
    }

    // 批量转移 ERC721 (辅助功能)
    // @param to: 接收地址
    // @param assets: NFT 资产列表
    function batchTransferERC721(
        address to,
        LibOrder.NFTInfo[] calldata assets
    ) external {
        // 遍历资产列表并逐一转移
        for (uint256 i = 0; i < assets.length; ++i) {
            IERC721(assets[i].collection).safeTransferNFT(
                _msgSender(), // 从调用者地址转移
                to,
                assets[i].tokenId
            );
        }
    }

    // 接收 NFT 的回调函数 (ERC721Receiver)
    // 必须实现此函数才能让合约接收 safeTransfer 发送的 NFT
    function onERC721Received(
        address,
        address,
        uint256,
        bytes memory
    ) public virtual returns (bytes4) {
        return this.onERC721Received.selector;
    }

    // 接收 ETH 的回调函数
    receive() external payable {}

    // 预留存储空间，支持未来合约升级
    uint256[50] private __gap;
}
