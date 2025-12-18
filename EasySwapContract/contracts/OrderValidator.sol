// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import {
    Initializable
} from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import {
    ContextUpgradeable
} from "@openzeppelin/contracts-upgradeable/utils/ContextUpgradeable.sol";
import {
    EIP712Upgradeable
} from "@openzeppelin/contracts-upgradeable/utils/cryptography/EIP712Upgradeable.sol";

import {Price} from "./libraries/RedBlackTreeLibrary.sol";
import {LibOrder, OrderKey} from "./libraries/LibOrder.sol";

/**
 * @title Verify the validity of the order parameters.
 */
/**
 * @title verify the validity of the order parameters.
 * @notice OrderValidator 是一个抽象合约，用于验证订单参数的合法性
 * 继承自 Initializable (用于代理模式初始化), ContextUpgradeable (用于获取上下文信息), EIP712Upgradeable (用于 EIP712 签名验证)
 */
abstract contract OrderValidator is
    Initializable,
    ContextUpgradeable,
    EIP712Upgradeable
{
    // EIP-1271 的魔术值，用于验证合约钱包签名是否有效
    bytes4 private constant EIP_1271_MAGIC_VALUE = 0x1626ba7e;

    // 订单取消状态的标记值 (使用 uint256 的最大值)
    // 当 filledAmount 的值为此常量时，表示该订单已被取消
    uint256 private constant CANCELLED = type(uint256).max;

    // 记录订单的填充状态：
    // Key: 订单哈希 (OrderKey)
    // Value: 已填充的数量
    //   - 对于 NFT 订单，1 表示已成交，0 表示未成交
    //   - 对于 ERC1155 或其他可部分成交的订单，表示已成交的数量
    //   - 如果 Value 等于 CANCELLED (type(uint256).max)，表示订单已取消
    mapping(OrderKey => uint256) public filledAmount;

    // OrderValidator 模块的初始化函数
    // @param EIP712Name: EIP712 域名称
    // @param EIP712Version: EIP712 版本号
    function __OrderValidator_init(
        string memory EIP712Name,
        string memory EIP712Version
    ) internal onlyInitializing {
        // 初始化上下文
        __Context_init();
        // 初始化 EIP712 签名验证模块
        __EIP712_init(EIP712Name, EIP712Version);
        // 执行无链式初始化
        __OrderValidator_init_unchained();
    }

    // 执行本合约特有的初始化逻辑
    function __OrderValidator_init_unchained() internal onlyInitializing {}

    /**
     * @notice 验证订单参数的有效性
     * @dev 检查 Maker 地址、过期时间、Salt 以及特定类型的必要参数
     * @param order: 待验证的订单结构体
     * @param isSkipExpiry: 是否跳过过期时间检查 (true 为跳过)
     */
    function _validateOrder(
        LibOrder.Order memory order,
        bool isSkipExpiry
    ) internal view {
        // 校验: 订单发起者 (Maker) 地址不能为空
        require(order.maker != address(0), "OVa: miss maker");

        // 检查过期时间（除非指定跳过检查）
        if (!isSkipExpiry) {
            // 校验: 订单必须未过期 (expiry == 0 表示永不过期)
            require(
                order.expiry == 0 || order.expiry > block.timestamp,
                "OVa: expired"
            );
        }
        // 校验: 盐值 (Salt) 不能为 0，Salt 用于确保同一用户的订单哈希唯一性
        require(order.salt != 0, "OVa: zero salt");

        // 根据买卖方向校验特定参数
        if (order.side == LibOrder.Side.List) {
            // 如果是卖单 (List)，必须指定有效的 NFT 合约地址
            require(
                order.nft.collection != address(0),
                "OVa: unsupported nft asset"
            );
        } else if (order.side == LibOrder.Side.Bid) {
            // 如果是买单 (Bid)，必须指定有效的出价金额 (Price > 0)
            require(Price.unwrap(order.price) > 0, "OVa: zero price");
        }
    }

    /**
     * @notice 获取订单的已填充数量
     * @param orderKey: 订单哈希
     * @return orderFilledAmount 返回该订单已填充的数量
     */
    function _getFilledAmount(
        OrderKey orderKey
    ) internal view returns (uint256 orderFilledAmount) {
        orderFilledAmount = filledAmount[orderKey];
        // 校验: 如果读取到的值为 CANCELLED，说明订单已取消，抛出异常
        require(orderFilledAmount != CANCELLED, "OVa: canceled");
    }

    /**
     * @notice 更新订单的填充状态
     * @param newAmount: 新的填充数量
     * @param orderKey: 订单哈希
     */
    function _updateFilledAmount(
        uint256 newAmount,
        OrderKey orderKey
    ) internal {
        // 校验: 确保不会意外将正常状态更新为 CANCELLED 状态（取消操作应调用 _cancelOrder）
        require(newAmount != CANCELLED, "OVa: canceled");
        // 更新 mapping 中的状态
        filledAmount[orderKey] = newAmount;
    }

    /**
     * @notice 取消订单
     * @dev 将订单状态标记为 CANCELLED (即 uint256.max)
     * @param orderKey: 要取消的订单哈希
     */
    function _cancelOrder(OrderKey orderKey) internal {
        // 将 filledAmount 设置为最大值，标记为已取消
        filledAmount[orderKey] = CANCELLED;
    }

    // 预留 50 个 uint256 大小的存储空间，支持未来合约升级而不破坏存储布局
    uint256[50] private __gap;
}
