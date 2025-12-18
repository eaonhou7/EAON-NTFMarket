import { Box, History, DollarSign, Wallet, LayoutGrid, List, TableProperties, Grid, Edit3, X, ExternalLink, Clock, TrendingUp } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useEffect, useState } from "react"
import { ListingDialog } from "./listingDailog"
import CollectionsApi from "@/api/collections"
import { useAccount } from "wagmi"
import { formatEther, parseEther } from "ethers"
import { SaleKind, Side, makeOrders } from "@/contracts/service/orderBookContract"
import { useEthersSigner } from "../../../hooks/useEthersSigner"

export function MainContent({myListOrders}: {myListOrders: any[]}) {

  const [listingDialogOpen, setListingDialogOpen] = useState(false);
  const [collections, setCollections] = useState<any[]>([]);
  const [canListNfts, setCanListNfts] = useState<any[]>([]);
  const [myNfts, setMyNfts] = useState<any[]>([]);
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [selectedNfts, setSelectedNfts] = useState<Set<string>>(new Set());
  const [prices, setPrices] = useState<{ [key: string]: string }>({});
  const [imageErrors, setImageErrors] = useState<Set<string>>(new Set());
  const [isListing, setIsListing] = useState(false);
  const [listingStatus, setListingStatus] = useState<string>('');
  const { address } = useAccount();
  const signer = useEthersSigner();

  async function loadCollections() {
    // @ts-ignore
    const {result} = await CollectionsApi.GetCollections({
      limit: 100,
      range: '7d',
    })

    setCollections(result);
    
  }

  async function loadMyNftsWithAlchemy() {
    if (!address) return;
    
    const apiKey = '8N8LPq7eV1mZPWkArtwvWHHB-EnADbER';
    const url = `https://eth-sepolia.g.alchemy.com/nft/v3/${apiKey}/getNFTsForOwner?owner=${address}&pageSize=100`;
    
    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: { 'accept': 'application/json' }
      });
      const {ownedNfts} = await response.json();
      console.log('=====', ownedNfts);
      
      setMyNfts(ownedNfts || []);
    } catch (err) {
      console.error('获取NFT失败:', err);
    }
  }

  useEffect(() => {
    if (address) {
      loadMyNftsWithAlchemy()
    }
  }, [address])

  useEffect(() => {
    loadCollections()
  }, [])

  // 过滤当前可以挂单的 NFT
  useEffect(() => {
    // 即使 collections 为空，也应该显示 NFT，或者至少显示所有从 Alchemy 获取到的 NFT
    // 这里暂时保持过滤逻辑，但增加大小写忽略，并打印日志以供调试
    if (myNfts?.length === 0) return;

    console.log('开始过滤 NFT...', { 
      totalNfts: myNfts.length, 
      collectionsCount: collections?.length 
    });

    // 如果 collections 为空（可能是加载失败或确实没有），暂时不过滤，显示所有 NFT
    // 或者根据业务需求，如果没有 collections 数据，可能无法判断是否可挂单
    
    const filteredNfts = myNfts.filter(it => {
      // 如果 collections 还没加载回来，或者为空，这里为了让用户看到刚 mint 的 NFT，
      // 可以选择暂时放行，或者严格依赖 collections。
      // 考虑到用户反馈，这里放宽条件：如果 collections 为空，或者匹配到了，都算作可以展示。
      // 但为了挂单安全，最好还是匹配 collection。
      
      // 1. 尝试匹配集合 (忽略大小写)
      const collection = collections?.find(c => 
        c.address?.toLowerCase() === it.contract?.address?.toLowerCase()
      );
      
      // 2. 如果找到了集合信息，或者这是刚才 mint 的那个合约地址（临时硬编码支持显示）
      const isTargetContract = it.contract?.address?.toLowerCase() === "0xbd8d85d9bdc8a07741e546bad7547d2907180781";
      
      if (collection || isTargetContract) {
        return true;
      }

      // 如果需要显示所有 NFT（只读），可以返回 true，但在挂单时可能会因为缺少 collection 信息报错
      // 这里先保持严格过滤，但加上特定合约的白名单
      return false;
    });

    console.log('过滤后的 NFT:', filteredNfts);
    setCanListNfts(filteredNfts);
  }, [collections, myNfts])

  function openListingDialog() {
    setListingDialogOpen(true);
  }

  function closeListingDialog() {
    setListingDialogOpen(false);
  }

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

  const handleBatchListing = async () => {
    if (!signer || !address) {
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
        maker: address,
        nft: {
          tokenId: nft.tokenId,
          collection: nft.contract.address,
          amount: 1
        },
        price: parseEther(nft.price),
        expiry: now,
        salt: Date.now() + index,
      }));

      console.log('准备创建的订单:', orders);
      setListingStatus('检查并授权NFT...');

      const result = await makeOrders(signer, orders, {
        autoApprove: true
      });
      
      console.log('挂单成功:', result);
      setListingStatus('挂单成功！');
      alert(`挂单成功！交易哈希: ${result.transactionHash}`);
      
      // 重置状态
      setSelectedNfts(new Set());
      setPrices({});
      
    } catch (error: any) {
      console.error('挂单失败:', error);
      let errorMessage = error.message;
      
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

  const handleCancelOrder = async (orderKey: string) => {
    // TODO: 实现取消挂单逻辑
    console.log('取消订单:', orderKey);
    alert('取消挂单功能开发中...');
  };

  const handleEditOrder = async (orderKey: string) => {
    // TODO: 实现编辑挂单逻辑
    console.log('编辑订单:', orderKey);
    alert('编辑挂单功能开发中...');
  };

  const formatTimeAgo = (timestamp: number) => {
    const now = Date.now();
    const diff = now - timestamp;
    const minutes = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    
    if (days > 0) return `${days}天前`;
    if (hours > 0) return `${hours}小时前`;
    if (minutes > 0) return `${minutes}分钟前`;
    return '刚刚';
  };

  const [tabIndex, setTabIndex] = useState(0)

  // 计算统计数据
  const totalListings = myListOrders.length;
  const totalValue = myListOrders.reduce((sum, order) => {
    try {
      return sum + parseFloat(formatEther(order.price || '0'));
    } catch {
      return sum;
    }
  }, 0);

  return (
    <div className="flex-1 p-4 bg-gray-900 min-h-screen">
      <ListingDialog canListNfts={canListNfts} collections={collections} open={listingDialogOpen} close={closeListingDialog} />

      {/* 页面头部 */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white">我的投资组合</h1>
          <p className="text-gray-400 mt-1">管理您的NFT资产和挂单</p>
        </div>
        <div className="flex items-center space-x-6">
          <div className="text-sm text-gray-400 space-x-6">
            <span className="inline-flex items-center">
              <span className="text-white font-medium">{totalListings}</span>
              <span className="ml-1">已挂单</span>
            </span>
            <span className="inline-flex items-center">
              <span className="text-white font-medium">{totalValue.toFixed(3)} ETH</span>
              <span className="ml-1">总估值</span>
            </span>
            <span className="inline-flex items-center">
              <TrendingUp className="h-4 w-4 text-green-400 mr-1" />
              <span className="text-green-400">+0.00%</span>
            </span>
          </div>
        </div>
      </div>

      {/* 标签页导航 */}
      <div className="mb-6">
        <div className="flex items-center space-x-4 border-b border-gray-700">
          <Button 
            variant="ghost" 
            className={`text-white border-b-2 ${tabIndex === 0 ? 'border-purple-400 text-purple-400' : 'border-transparent'} rounded-none hover:text-purple-400`} 
            onClick={() => setTabIndex(0)}
          >
            <Box className="h-4 w-4 mr-2" />
            库存 ({myNfts.length})
          </Button>
          <Button 
            variant="ghost" 
            className={`text-white border-b-2 ${tabIndex === 1 ? 'border-purple-400 text-purple-400' : 'border-transparent'} rounded-none hover:text-purple-400`} 
            onClick={() => setTabIndex(1)}
          >
            <History className="h-4 w-4 mr-2" />
            挂单记录 ({totalListings})
          </Button>
          <Button 
            variant="ghost" 
            className={`text-white border-b-2 ${tabIndex === 2 ? 'border-purple-400 text-purple-400' : 'border-transparent'} rounded-none hover:text-purple-400`} 
            onClick={() => setTabIndex(2)}
          >
            <DollarSign className="h-4 w-4 mr-2" />
            出价记录
          </Button>
          <Button 
            variant="ghost" 
            className={`text-white border-b-2 ${tabIndex === 3 ? 'border-purple-400 text-purple-400' : 'border-transparent'} rounded-none hover:text-purple-400`} 
            onClick={() => setTabIndex(3)}
          >
            <Wallet className="h-4 w-4 mr-2" />
            交易历史
          </Button>
        </div>
      </div>

      {/* 库存页面内容 */}
      {tabIndex === 0 && (
        <div>
          {/* 视图控制和操作栏 */}
          <div className="flex justify-between items-center mb-6">
            <div className="flex items-center space-x-4">
              <h2 className="text-lg font-semibold text-white">我的NFT库存</h2>
              {canListNfts.length > 0 && (
                <span className="px-2 py-1 bg-blue-600/20 text-blue-400 text-xs rounded-full">
                  {canListNfts.length} 个可挂单NFT
                </span>
              )}
              {selectedNfts.size > 0 && (
                <span className="px-2 py-1 bg-purple-600/20 text-purple-400 text-xs rounded-full">
                  已选择 {selectedNfts.size} 个
                </span>
              )}
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setViewMode('grid')}
                className={`${viewMode === 'grid' ? 'bg-gray-700 text-white' : 'text-gray-400 hover:text-white'}`}
              >
                <LayoutGrid className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setViewMode('list')}
                className={`${viewMode === 'list' ? 'bg-gray-700 text-white' : 'text-gray-400 hover:text-white'}`}
              >
                <List className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* NFT库存列表 */}
          {canListNfts.length === 0 ? (
            // <div>const arr = [];
            <div className="text-center py-16">
              <div className="text-6xl mb-4">🎨</div>
              <h3 className="text-xl font-semibold text-white mb-2">暂无可挂单的NFT</h3>
              <p className="text-gray-400 mb-6">请确保您拥有支持的NFT集合</p>
            </div>
          ) : (
            <div className={`${viewMode === 'grid' 
              ? 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4' 
              : 'space-y-4'
            }`}>
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
                    } ${viewMode === 'grid' ? 'p-4' : 'p-4 flex items-center space-x-4'}`}
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
                    <div className={`${viewMode === 'grid' ? 'aspect-square mb-4' : 'w-16 h-16 flex-shrink-0'} rounded-lg overflow-hidden bg-gray-700 relative`}>
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
                            <div className={`${viewMode === 'grid' ? 'text-4xl mb-2' : 'text-2xl'}`}>🖼️</div>
                            {viewMode === 'grid' && <div className="text-xs">图片加载失败</div>}
                          </div>
                        </div>
                      )}
                    </div>
                    
                    {/* NFT 信息 */}
                    <div className={`${viewMode === 'grid' ? 'space-y-3' : 'flex-1 space-y-2'}`}>
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
                      
                      {/* 描述 - 仅在网格视图显示 */}
                      {viewMode === 'grid' && nft.description && (
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
                      
                      {/* 合约信息 - 仅在网格视图显示 */}
                      {viewMode === 'grid' && (
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
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {/* 批量操作面板 */}
          {selectedNfts.size > 0 && (
            <div className="fixed bottom-20 left-1/2 transform -translate-x-1/2 bg-gray-800 border border-gray-600 rounded-xl p-4 shadow-2xl">
              <div className="flex items-center space-x-4">
                <span className="text-white font-medium">
                  已选择 {selectedNfts.size} 个NFT
                </span>
                <Button
                  variant="outline"
                  onClick={() => {
                    setSelectedNfts(new Set());
                    setPrices({});
                  }}
                  className="border-gray-600 text-gray-300 hover:bg-gray-700"
                >
                  清空选择
                </Button>
                <Button
                  onClick={handleBatchListing}
                  disabled={isListing}
                  className="bg-purple-600 hover:bg-purple-700 text-white"
                >
                  {isListing 
                    ? (listingStatus || '挂单中...') 
                    : `批量挂单 (${selectedNfts.size})`
                  }
                </Button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* 挂单记录页面内容 */}
      {tabIndex === 1 && (
        <div>
          {/* 视图控制和操作栏 */}
          <div className="flex justify-between items-center mb-6">
            <div className="flex items-center space-x-4">
              <h2 className="text-lg font-semibold text-white">挂单记录</h2>
              {myListOrders.length > 0 && (
                <span className="px-2 py-1 bg-purple-600/20 text-purple-400 text-xs rounded-full">
                  {myListOrders.length} 个活跃订单
                </span>
              )}
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setViewMode('grid')}
                className={`${viewMode === 'grid' ? 'bg-gray-700 text-white' : 'text-gray-400 hover:text-white'}`}
              >
                <LayoutGrid className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setViewMode('list')}
                className={`${viewMode === 'list' ? 'bg-gray-700 text-white' : 'text-gray-400 hover:text-white'}`}
              >
                <List className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* 挂单列表 */}
          {myListOrders.length === 0 ? (
            <div className="text-center py-16">
              <div className="text-6xl mb-4">📝</div>
              <h3 className="text-xl font-semibold text-white mb-2">暂无挂单记录</h3>
              <p className="text-gray-400 mb-6">您还没有创建任何挂单，开始挂单来出售您的NFT</p>
              <Button 
                onClick={() => setTabIndex(0)}
                className="bg-purple-600 hover:bg-purple-700 text-white"
              >
                去库存页面挂单
              </Button>
            </div>
          ) : (
            <div className={`${viewMode === 'grid' 
              ? 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6' 
              : 'space-y-4'
            }`}>
              {myListOrders.map((order) => (
                <div
                  key={order.collection_address + order.token_id}
                  className={`bg-gray-800 rounded-xl border border-gray-700 hover:border-purple-400 transition-all duration-200 ${
                    viewMode === 'grid' ? 'p-4' : 'p-4 flex items-center space-x-4'
                  }`}
                >
                  {/* NFT 图片 */}
                  <div className={`${viewMode === 'grid' ? 'aspect-square mb-4' : 'w-16 h-16 flex-shrink-0'} rounded-lg overflow-hidden bg-gray-700`}>
                    <img 
                      src={order.collection_image_uri || order.item_image_uri} 
                      alt={order.item_name || 'NFT'} 
                      className="w-full h-full object-cover"
                      onError={(e) => {
                        e.currentTarget.src = 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAwIiBoZWlnaHQ9IjIwMCIgdmlld0JveD0iMCAwIDIwMCAyMDAiIGZpbGw9Im5vbmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+CjxyZWN0IHdpZHRoPSIyMDAiIGhlaWdodD0iMjAwIiBmaWxsPSIjMzc0MTUxIi8+CjxwYXRoIGQ9Ik0xMDAgNTBMMTUwIDEyNUg1MEwxMDAgNTBaIiBmaWxsPSIjNkI3Mjg1Ii8+Cjx0ZXh0IHg9IjEwMCIgeT0iMTYwIiBmb250LWZhbWlseT0ic2Fucy1zZXJpZiIgZm9udC1zaXplPSIxMiIgZmlsbD0iIzlDQTNBRiIgdGV4dC1hbmNob3I9Im1pZGRsZSI+TkZUPC90ZXh0Pgo8L3N2Zz4K';
                      }}
                    />
                  </div>

                  {/* NFT 信息 */}
                  <div className={`${viewMode === 'grid' ? '' : 'flex-1'}`}>
                    <div className="flex items-start justify-between mb-2">
                      <div>
                        <h3 className="font-semibold text-white truncate">
                          {order.item_name || `#${order.token_id}`}
                        </h3>
                        <p className="text-sm text-gray-400 truncate">
                          {order.collection_name}
                        </p>
                      </div>
                      {viewMode === 'grid' && (
                        <span className="px-2 py-1 bg-green-600/20 text-green-400 text-xs rounded-full">
                          活跃
                        </span>
                      )}
                    </div>

                    {/* 价格和状态 */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-400">挂单价格</span>
                        <span className="font-semibold text-white">
                          {(() => {
                            try {
                              return `${parseFloat(formatEther(order.price || '0')).toFixed(3)} ETH`;
                            } catch {
                              return '0.000 ETH';
                            }
                          })()}
                        </span>
                      </div>
                      
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-400">创建时间</span>
                        <span className="text-sm text-gray-300 flex items-center">
                          <Clock className="h-3 w-3 mr-1" />
                          {formatTimeAgo(order.created_at || Date.now())}
                        </span>
                      </div>

                      {viewMode === 'list' && (
                        <span className="inline-flex px-2 py-1 bg-green-600/20 text-green-400 text-xs rounded-full">
                          活跃
                        </span>
                      )}
                    </div>

                    {/* 操作按钮 */}
                    <div className={`${viewMode === 'grid' ? 'mt-4' : 'mt-2'} flex gap-2`}>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleEditOrder(order.order_key)}
                        className="flex-1 border-gray-600 text-gray-300 hover:bg-gray-700 hover:text-white"
                      >
                        <Edit3 className="h-3 w-3 mr-1" />
                        编辑
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleCancelOrder(order.order_key)}
                        className="flex-1 border-red-600/50 text-red-400 hover:bg-red-600/10 hover:border-red-500"
                      >
                        <X className="h-3 w-3 mr-1" />
                        取消
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => window.open(`https://sepolia.etherscan.io/address/${order.collection_address}`, '_blank')}
                        className="p-2 text-gray-400 hover:text-white"
                      >
                        <ExternalLink className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* 其他标签页内容 */}
      {tabIndex === 2 && (
        <div className="text-center py-16">
          <div className="text-6xl mb-4">💰</div>
          <h3 className="text-xl font-semibold text-white mb-2">出价记录</h3>
          <p className="text-gray-400">此功能正在开发中...</p>
        </div>
      )}

      {tabIndex === 3 && (
        <div className="text-center py-16">
          <div className="text-6xl mb-4">📈</div>
          <h3 className="text-xl font-semibold text-white mb-2">交易历史</h3>
          <p className="text-gray-400">此功能正在开发中...</p>
        </div>
      )}

      {/* 底部操作栏 */}
      <div className="fixed bottom-0 left-0 right-0 border-t border-gray-700 bg-gray-900/95 backdrop-blur-sm p-4">
        <div className="flex justify-center space-x-4">
          <Button 
            className="bg-purple-600 hover:bg-purple-700 text-white" 
            onClick={() => setTabIndex(0)}
          >
            库存管理 ({canListNfts.length})
          </Button>
          <Button 
            className="bg-blue-600 hover:bg-blue-700 text-white"
            onClick={() => setTabIndex(1)}
          >
            挂单记录 ({totalListings})
          </Button>
          <Button className="bg-green-600 hover:bg-green-700 text-white">
            更多功能
          </Button>
        </div>
      </div>
    </div>
  )
}

