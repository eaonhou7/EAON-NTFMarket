// WalletLogin.tsx
import React, { useState, useEffect } from "react";
import { useAccount, useDisconnect } from "wagmi";
import { useConnectModal } from "@rainbow-me/rainbowkit";
import { formatAddress } from "@/lib/utils"; // 假设我们有这个工具函数来格式化地址

const WalletLogin = () => {
  const { address, isConnected } = useAccount();
  const { disconnect } = useDisconnect();
  const { openConnectModal } = useConnectModal();


  return (
    <div className="flex items-center gap-2">
      {isConnected && address ? (
        <div className="flex items-center gap-2">
          <span className="text-sm">{formatAddress(address)}</span>
          <button
            onClick={() => disconnect()}
            className="px-4 py-2 text-sm rounded-lg bg-red-500 text-white hover:bg-red-600"
          >
            断开连接
          </button>
        </div>
      ) : (
        <button
          onClick={openConnectModal}
          className="px-4 py-2 text-sm rounded-lg bg-blue-500 text-white hover:bg-blue-600"
        >
          连接钱包
        </button>
      )}
    </div>
  );
};

export default WalletLogin;
