import {
  Dialog,
  DialogDescription,
  DialogPanel,
  DialogTitle,
  Transition,
} from "@headlessui/react";
import { Fragment, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { X, ExternalLink } from "lucide-react";
import { SaleKind, Side, makeOrders } from "@/contracts/service/orderBookContract";
import { useAccount } from 'wagmi'
import { parseEther } from "ethers";
import { useEthersSigner } from "../../../hooks/useEthersSigner";

export function ListingNftDialog({
  open,
  close,
  canListNfts
}: {
  open: boolean;
  close: () => void;
  canListNfts: any[];
}) {
  const { address: owner } = useAccount();
  const signer = useEthersSigner();
  const [selectedNfts, setSelectedNfts] = useState<Set<string>>(new Set());
  const [prices, setPrices] = useState<{ [key: string]: string }>({});
  const [imageErrors, setImageErrors] = useState<Set<string>>(new Set());
  const [isListing, setIsListing] = useState(false);
  const [listingStatus, setListingStatus] = useState<string>('');

  useEffect(() => {
    if (open) {
     
    }
  }, [open]);

  const toggleNftSelection = (nftKey: string) => {
    const newSelected = new Set(selectedNfts);
    if (newSelected.has(nftKey)) {
      newSelected.delete(nftKey);
    } else {
      newSelected.add(nftKey);
    }
    setSelectedNfts(newSelected);
  };

  const updatePrice = (nftKey: string, price: string) => {
    setPrices(prev => ({
      ...prev,
      [nftKey]: price
    }));
  };

  const handleImageError = (nftKey: string) => {
    setImageErrors(prev => new Set([...prev, nftKey]));
  };

  const handleListSelected = async () => {
    if (!signer || !owner) {
      alert('请先连接钱包');
      return;
    }

    const selectedNftData = canListNfts.filter(nft => {
      const nftKey = `${nft.contract.address}-${nft.tokenId}`;
      return selectedNfts.has(nftKey);
    }).map(nft => ({
      ...nft,
      price: prices[`${nft.contract.address}-${nft.tokenId}`] || '0'
    }));

    // 验证价格
    const invalidPrices = selectedNftData.filter(nft => 
      !nft.price || parseFloat(nft.price) <= 0
    );
    
    if (invalidPrices.length > 0) {
      alert('请为所有选中的NFT设置有效的价格');
      return;
    }

    try {
      setIsListing(true);
      setListingStatus('准备创建订单...');
      
      const now = Math.floor(Date.now() / 1000) + 100000;
      const orders = selectedNftData.map((nft, index) => ({
        side: Side.List,
        saleKind: SaleKind.FixedPriceForItem,
        maker: owner,
        nft: {
          tokenId: nft.tokenId,
          collection: nft.contract.address,
          amount: 1
        },
        price: parseEther(nft.price),  // 保持为bigint类型
        expiry: now,
        salt: Date.now() + index, // 使用时间戳 + 索引确保唯一性
      }));

      console.log('准备创建的订单:', orders);
      setListingStatus('检查并授权NFT...');

      const result = await makeOrders(signer, orders, {
        autoApprove: true  // 启用自动授权NFT
      });
      
      console.log('挂单成功:', result);
      setListingStatus('挂单成功！');
      alert(`挂单成功！交易哈希: ${result.transactionHash}`);
      
      // 重置状态
      setSelectedNfts(new Set());
      setPrices({});
      close();
      
    } catch (error: any) {
      console.error('挂单失败:', error);
      let errorMessage = error.message;
      
      // 根据错误类型提供更友好的提示
      if (errorMessage.includes('missing revert data')) {
        errorMessage = '交易可能会失败，请检查：\n1. NFT是否属于您\n2. 价格是否有效\n3. 网络是否正确\n4. 钱包余额是否足够支付gas费';
      } else if (errorMessage.includes('user rejected')) {
        errorMessage = '用户取消了交易';
      } else if (errorMessage.includes('insufficient funds')) {
        errorMessage = 'gas费不足，请确保钱包有足够的ETH';
      }
      
      alert(`挂单失败: ${errorMessage}`);
    } finally {
      setIsListing(false);
      setListingStatus('');
    }
  };

  return (
    <Dialog as="div" className="relative z-50" open={open} onClose={close}>
      <Transition.Child
        as={Fragment}
        enter="ease-out duration-300"
        enterFrom="opacity-0"
        enterTo="opacity-100"
        leave="ease-in duration-200"
        leaveFrom="opacity-100"
        leaveTo="opacity-0"
      >
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm" />
      </Transition.Child>

      <div className="fixed inset-0 overflow-y-auto">
        <div className="flex min-h-full items-center justify-center p-4">
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0 scale-95"
            enterTo="opacity-100 scale-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100 scale-100"
            leaveTo="opacity-0 scale-95"
          >
            <DialogPanel className="w-full max-w-4xl transform overflow-hidden rounded-2xl bg-gradient-to-br from-gray-900 to-gray-800 text-white shadow-2xl transition-all border border-gray-700">
              {/* Header */}
              <div className="flex items-center justify-between p-6 border-b border-gray-700">
                <div>
                  <DialogTitle className="text-2xl font-bold text-white">
                    选择要挂单的 NFT
                  </DialogTitle>
                  <DialogDescription className="text-gray-400 mt-1">
                    选择NFT并设置价格进行挂单
                  </DialogDescription>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={close}
                  className="text-gray-400 hover:text-white hover:bg-gray-700"
                >
                  <X className="h-5 w-5" />
                </Button>
              </div>

              {/* Content */}
              <div className="p-6 max-h-[70vh] overflow-y-auto">
                {canListNfts.length === 0 ? (
                  <div className="text-center py-12 text-gray-400">
                    <div className="text-6xl mb-4">🎨</div>
                    <p className="text-lg">暂无可挂单的NFT</p>
                    <p className="text-sm mt-2">请确保您拥有支持的NFT集合</p>
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {canListNfts.map(nft => {
                      const nftKey = `${nft.contract.address}-${nft.tokenId}`;
                      const isSelected = selectedNfts.has(nftKey);
                      
                      return (
                        <div
                          key={nftKey}
                          className={`relative rounded-xl border-2 transition-all duration-200 cursor-pointer ${
                            isSelected 
                              ? 'border-purple-400 bg-purple-900/20 shadow-lg shadow-purple-400/20' 
                              : 'border-gray-600 bg-gray-800/50 hover:border-gray-500'
                          }`}
                          onClick={() => toggleNftSelection(nftKey)}
                        >
                          {/* 选中状态指示器 */}
                          {isSelected && (
                            <div className="absolute top-3 right-3 z-10">
                              <div className="w-6 h-6 bg-purple-500 rounded-full flex items-center justify-center">
                                <div className="w-3 h-3 bg-white rounded-full" />
                              </div>
                            </div>
                          )}
                          
                          {/* NFT 图片 */}
                          <div className="aspect-square rounded-t-xl overflow-hidden bg-gray-700 relative">
                            {!imageErrors.has(nftKey) && (
                              <img 
                                src={nft.image?.cachedUrl || nft.image?.originalUrl} 
                                alt={nft.name || `Token #${nft.tokenId}`}
                                className="w-full h-full object-cover transition-transform duration-200 hover:scale-105"
                                onError={() => handleImageError(nftKey)}
                              />
                            )}
                            {/* 占位图片 */}
                            {imageErrors.has(nftKey) && (
                              <div className="absolute inset-0 bg-gray-700 flex items-center justify-center text-gray-400">
                                <div className="text-center">
                                  <div className="text-4xl mb-2">🖼️</div>
                                  <div className="text-xs">图片加载失败</div>
                                </div>
                              </div>
                            )}
                          </div>
                          
                          {/* NFT 信息 */}
                          <div className="p-4 space-y-3">
                            <div>
                              <h3 className="font-semibold text-white truncate">
                                {nft.name || `${nft.contract.name} #${nft.tokenId}`}
                              </h3>
                              <div className="flex items-center gap-2 mt-1">
                                <span className="text-xs bg-gray-700 text-gray-300 px-2 py-1 rounded-md">
                                  {nft.contract.symbol}
                                </span>
                                <span className="text-xs text-gray-400">#{nft.tokenId}</span>
                              </div>
                            </div>
                            
                            {/* 描述 */}
                            {nft.description && (
                              <p className="text-sm text-gray-400 line-clamp-2">
                                {nft.description}
                              </p>
                            )}
                            
                            {/* 价格输入 */}
                            <div className="space-y-2">
                              <label className="text-sm font-medium text-gray-300">
                                挂单价格 (ETH)
                              </label>
                              <Input
                                type="number"
                                step="0.001"
                                min="0"
                                placeholder="0.000"
                                value={prices[nftKey] || ''}
                                onChange={(e) => updatePrice(nftKey, e.target.value)}
                                onClick={(e) => e.stopPropagation()}
                                className="bg-gray-700 border-gray-600 text-white placeholder-gray-400 focus:border-purple-400"
                              />
                            </div>
                            
                            {/* 合约信息 */}
                            <div className="flex items-center justify-between text-xs text-gray-500">
                              <span className="truncate flex-1">
                                {nft.contract.address.slice(0, 6)}...{nft.contract.address.slice(-4)}
                              </span>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  window.open(`https://sepolia.etherscan.io/address/${nft.contract.address}`, '_blank');
                                }}
                                className="h-6 w-6 p-0 text-gray-500 hover:text-gray-300"
                              >
                                <ExternalLink className="h-3 w-3" />
                              </Button>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>

              {/* Footer */}
              {canListNfts.length > 0 && (
                <div className="flex items-center justify-between p-6 border-t border-gray-700 bg-gray-800/50">
                  <div className="text-sm text-gray-400">
                    已选择 {selectedNfts.size} 个NFT
                  </div>
                  <div className="flex gap-3">
                    <Button
                      variant="outline"
                      onClick={close}
                      className="border-gray-600 text-gray-300 hover:bg-gray-700"
                    >
                      取消
                    </Button>
                    <Button
                      onClick={handleListSelected}
                      disabled={selectedNfts.size === 0 || isListing}
                      className="bg-purple-600 hover:bg-purple-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {isListing 
                        ? (listingStatus || '挂单中...') 
                        : `挂单 ${selectedNfts.size > 0 ? `(${selectedNfts.size})` : ''}`
                      }
                    </Button>
                  </div>
                </div>
              )}
            </DialogPanel>
          </Transition.Child>
        </div>
      </div>
    </Dialog>
  );
}
