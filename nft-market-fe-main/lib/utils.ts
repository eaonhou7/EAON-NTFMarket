import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"
 
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const formatAddress = (address: string): string => {
  return `${address.slice(0, 6)}...${address.slice(-4)}`;
};

/**
 * 根据链 ID 获取区块链浏览器的基础 URL
 */
export const getExplorerBaseUrl = (chainId: string | number): string => {
  const id = typeof chainId === 'string' ? parseInt(chainId) : chainId;
  
  switch (id) {
    case 1: // Ethereum Mainnet
      return 'https://etherscan.io';
    case 10: // Optimism
      return 'https://optimistic.etherscan.io';
    case 11155111: // Sepolia Testnet
      return 'https://sepolia.etherscan.io';
    default:
      // 默认使用 Sepolia（测试环境）
      return 'https://sepolia.etherscan.io';
  }
};

/**
 * 生成 NFT 在区块链浏览器中的链接
 */
export const getNftExplorerUrl = (
  chainId: string | number,
  contractAddress: string,
  tokenId: string
): string => {
  const baseUrl = getExplorerBaseUrl(chainId);
  return `${baseUrl}/nft/${contractAddress}/${tokenId}`;
};

/**
 * 根据地址或字符串生成头像 URL
 * 使用 Dicebear API 生成基于 seed 的头像
 */
export const getAvatarUrl = (addressOrSeed: string): string => {
  if (!addressOrSeed) {
    // 如果没有地址，生成一个随机 seed
    addressOrSeed = Math.random().toString(36).substring(7);
  }
  
  // 使用 Dicebear 的 avataaars 风格，基于地址生成
  // 同一个地址总是生成相同的头像
  return `https://api.dicebear.com/7.x/avataaars/svg?seed=${encodeURIComponent(addressOrSeed)}`;
};

