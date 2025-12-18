import { getDefaultConfig } from '@rainbow-me/rainbowkit';
import { http } from 'wagmi';
import {
  mainnet,
  optimism,
  polygon,
  sepolia,
} from 'wagmi/chains';
// console.log(process.env.WalletConnectId)


export const config = getDefaultConfig({
  appName: 'RainbowKit App',
  projectId: '94066ab3be2718981f226c7407038ba4',
  chains: [
    mainnet,
    sepolia,
    // polygon,
    // optimism,
  ],
  transports: {
    // 替换之前 不可用的 https://rpc.sepolia.org/
    [mainnet.id]: http('https://mainnet.infura.io/v3/d8ed0bd1de8242d998a1405b6932ab33'),
    // [polygon.id]: http('https://polygon.infura.io/v3/d8ed0bd1de8242d998a1405b6932ab33'),
    // [optimism.id]: http('https://optimism.infura.io/v3/d8ed0bd1de8242d998a1405b6932ab33'),
    [sepolia.id]: http('https://sepolia.infura.io/v3/d8ed0bd1de8242d998a1405b6932ab33')
  },
  ssr: true,
});