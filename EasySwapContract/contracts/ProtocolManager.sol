// SPDX-License-Identifier: MIT

pragma solidity ^0.8.19;

import {
    Initializable
} from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {LibPayInfo} from "./libraries/LibPayInfo.sol";

// 定义 ProtocolManager 抽象合约，继承自 Initializable（用于代理合约初始化）和 OwnableUpgradeable（提供 owner 权限控制）
abstract contract ProtocolManager is Initializable, OwnableUpgradeable {
    // 协议手续费分成比例，单位为万分位 (Example: 250 = 2.5%)
    // 此变量存储了当前协议收取的交易手续费比例
    uint128 public protocolShare;

    // 当协议手续费比例更新时触发的事件
    // @param newProtocolShare: 新设置的协议手续费比例
    event LogUpdatedProtocolShare(uint128 indexed newProtocolShare);

    // ProtocolManager 模块的初始化函数
    // 仅在合约部署后的初始化阶段调用一次 (onlyInitializing)
    // @param newProtocolShare: 初始设置的协议手续费比例
    function __ProtocolManager_init(
        uint128 newProtocolShare
    ) internal onlyInitializing {
        // __Ownable_init(_msgSender());
        // 调用无链式初始化函数执行具体逻辑
        __ProtocolManager_init_unchained(newProtocolShare);
    }

    // 执行本合约特有的初始化逻辑，不包含父合约初始化
    // @param newProtocolShare: 初始设置的协议手续费比例
    function __ProtocolManager_init_unchained(
        uint128 newProtocolShare
    ) internal onlyInitializing {
        // 调用内部函数设置初始的协议手续费比例
        _setProtocolShare(newProtocolShare);
    }

    // 外部调用的设置协议手续费比例函数
    // 仅限合约所有者 (owner) 可以调用 (onlyOwner 修饰符)
    // @param newProtocolShare: 想要设置的新手续费比例
    function setProtocolShare(uint128 newProtocolShare) external onlyOwner {
        // 调用内部函数执行实际的设置逻辑
        _setProtocolShare(newProtocolShare);
    }

    // 内部使用的设置协议手续费比例函数，包含校验逻辑
    // @param newProtocolShare: 新的手续费比例
    function _setProtocolShare(uint128 newProtocolShare) internal {
        // 校验：新设置的比例不能超过 LibPayInfo 中定义的最大比例 (MAX_PROTOCOL_SHARE)
        // 如果超过，则抛出 "PM: exceed max protocol share" 错误
        require(
            newProtocolShare <= LibPayInfo.MAX_PROTOCOL_SHARE,
            "PM: exceed max protocol share"
        );
        // 更新状态变量 protocolShare 为新值
        protocolShare = newProtocolShare;
        // 触发 LogUpdatedProtocolShare 事件，通知链下监听者费率已变更
        emit LogUpdatedProtocolShare(newProtocolShare);
    }

    // 预留 50 个 uint256 大小的存储空间
    // 这是升级合约的最佳实践，用于防止未来升级合约添加新变量时导致存储槽位冲突
    uint256[50] private __gap;
}
