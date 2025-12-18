// 转移合约所有权的脚本
// 需要用合约 owner 地址执行

import { ethers } from "ethers";
import MetaNodeNFTABI from "../contracts/abis/MetaNodeNFT.json";

async function transferOwnership() {
  // 连接到 Sepolia 测试网
  const provider = new ethers.JsonRpcProvider(
    "https://sepolia.infura.io/v3/YOUR_INFURA_KEY"
  );

  // 使用合约 owner 的私钥（注意：不要提交到 Git）
  const ownerPrivateKey = "YOUR_OWNER_PRIVATE_KEY";
  const signer = new ethers.Wallet(ownerPrivateKey, provider);

  // 新的 owner 地址
  const newOwner = "0x885A3fDd294dBA8fE7FBB51314d4f4686611389F";

  // 连接合约
  const contract = new ethers.Contract(
    MetaNodeNFTABI.address,
    MetaNodeNFTABI.abi,
    signer
  );

  console.log("当前 owner:", await contract.owner());
  console.log("转移所有权到:", newOwner);

  // 转移所有权
  const tx = await contract.transferOwnership(newOwner);
  console.log("交易哈希:", tx.hash);

  await tx.wait();
  console.log("所有权转移成功！");
  console.log("新的 owner:", await contract.owner());
}

transferOwnership().catch(console.error);

