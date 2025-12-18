"use client";

import { formatUnits } from "ethers";
import { ExternalLink, Copy, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { formatAddress, getNftExplorerUrl } from "@/lib/utils";
import { useState } from "react";

const ETH_DECIMALS = 18;

interface ItemDetailViewProps {
  itemDetail: any;
  chainId?: string | number;
}

export function ItemDetailView({ itemDetail, chainId = 11155111 }: ItemDetailViewProps) {
  const [copied, setCopied] = useState<string | null>(null);
  const [imageError, setImageError] = useState(false);
  const [videoError, setVideoError] = useState(false);

  const formatPrice = (price: string | number, decimals: number = ETH_DECIMALS): string => {
    if (!price || price === "0" || price === 0) return "-";
    try {
      const priceStr = typeof price === "string" ? price : price.toString();
      const ethValue = formatUnits(priceStr, decimals);
      const numValue = parseFloat(ethValue);
      if (numValue === 0) return "-";
      if (numValue >= 1) {
        return `${numValue.toFixed(4)} ETH`;
      } else if (numValue >= 0.01) {
        return `${numValue.toFixed(6)} ETH`;
      } else {
        return `${numValue.toFixed(8)} ETH`;
      }
    } catch {
      return "-";
    }
  };

  const formatTime = (timestamp: number): string => {
    if (!timestamp || timestamp === 0) return "-";
    const date = new Date(timestamp * 1000);
    return date.toLocaleString("zh-CN");
  };

  const copyToClipboard = async (text: string, key: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(key);
      setTimeout(() => setCopied(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  const hasListing = itemDetail.list_order_id && itemDetail.list_order_id !== "";
  const hasBid = itemDetail.bid_order_id && itemDetail.bid_order_id !== "";

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* 左侧：图片/视频区域 */}
        <div className="space-y-4">
          <div className="relative aspect-square w-full rounded-xl border border-border bg-muted overflow-hidden">
            {!imageError && itemDetail.image_uri ? (
              <img
                src={itemDetail.image_uri}
                alt={itemDetail.name || `Token #${itemDetail.token_id}`}
                className="w-full h-full object-cover"
                onError={() => setImageError(true)}
              />
            ) : (
              <div className="absolute inset-0 bg-muted flex items-center justify-center">
                <div className="text-center">
                  <div className="text-6xl mb-2">🖼️</div>
                  <div className="text-sm text-muted-foreground">图片加载失败</div>
                </div>
              </div>
            )}
          </div>

          {/* 视频（如果有） */}
          {itemDetail.video_uri && !videoError && (
            <div className="relative aspect-video w-full rounded-xl border border-border bg-muted overflow-hidden">
              <video
                src={itemDetail.video_uri}
                controls
                className="w-full h-full object-cover"
                onError={() => setVideoError(true)}
              />
            </div>
          )}

          {/* 区块链浏览器链接 */}
          <Button
            variant="outline"
            className="w-full"
            onClick={() => {
              const url = getNftExplorerUrl(
                chainId,
                itemDetail.collection_address,
                itemDetail.token_id
              );
              window.open(url, "_blank", "noopener,noreferrer");
            }}
          >
            <ExternalLink className="h-4 w-4 mr-2" />
            在区块链浏览器中查看
          </Button>
        </div>

        {/* 右侧：详细信息 */}
        <div className="space-y-6">
          {/* 标题和基本信息 */}
          <div className="space-y-4">
            <div>
              <h1 className="text-3xl font-bold mb-2">
                {itemDetail.name || `${itemDetail.collection_name || "NFT"} #${itemDetail.token_id}`}
              </h1>
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-muted-foreground">Token ID: #{itemDetail.token_id}</span>
                {itemDetail.rarity_rank && itemDetail.rarity_rank > 0 && (
                  <span className="text-sm bg-primary/10 text-primary px-3 py-1 rounded-full">
                    稀有度排名: #{itemDetail.rarity_rank}
                  </span>
                )}
                {itemDetail.rarity_value && (
                  <span className="text-sm bg-secondary/50 text-secondary-foreground px-3 py-1 rounded-full">
                    稀有度值: {itemDetail.rarity_value.toFixed(2)}
                  </span>
                )}
              </div>
            </div>

            {/* 集合信息 */}
            {itemDetail.collection_name && (
              <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/50">
                {itemDetail.collection_image_uri && (
                  <img
                    src={itemDetail.collection_image_uri}
                    alt={itemDetail.collection_name}
                    className="w-12 h-12 rounded-full object-cover"
                  />
                )}
                <div>
                  <div className="text-sm text-muted-foreground">集合</div>
                  <div className="font-semibold">{itemDetail.collection_name}</div>
                </div>
              </div>
            )}
          </div>

          {/* 价格信息卡片 */}
          <div className="grid grid-cols-2 gap-4">
            {itemDetail.floor_price && itemDetail.floor_price !== "0" && (
              <div className="p-4 rounded-lg border border-border bg-card">
                <div className="text-sm text-muted-foreground mb-1">地板价</div>
                <div className="text-xl font-bold">{formatPrice(itemDetail.floor_price)}</div>
              </div>
            )}
            {itemDetail.last_sell_price && itemDetail.last_sell_price !== "0" && (
              <div className="p-4 rounded-lg border border-border bg-card">
                <div className="text-sm text-muted-foreground mb-1">最后成交价</div>
                <div className="text-xl font-bold">{formatPrice(itemDetail.last_sell_price)}</div>
              </div>
            )}
          </div>

          {/* 挂单信息 */}
          {hasListing && (
            <div className="p-4 rounded-lg border border-green-500/20 bg-green-500/5">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold text-green-500">当前挂单</span>
                  <span className="text-xs bg-green-500/20 text-green-500 px-2 py-0.5 rounded">
                    可购买
                  </span>
                </div>
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">挂单价格</span>
                  <span className="text-lg font-bold text-green-500">
                    {formatPrice(itemDetail.list_price)}
                  </span>
                </div>
                {itemDetail.list_time && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">挂单时间</span>
                    <span className="text-sm">{formatTime(itemDetail.list_time)}</span>
                  </div>
                )}
                {itemDetail.list_expire_time && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">过期时间</span>
                    <span className="text-sm">{formatTime(itemDetail.list_expire_time)}</span>
                  </div>
                )}
                {itemDetail.list_maker && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">挂单者</span>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-mono">{formatAddress(itemDetail.list_maker)}</span>
                      <button
                        onClick={() => copyToClipboard(itemDetail.list_maker, "list_maker")}
                        className="text-muted-foreground hover:text-foreground"
                      >
                        {copied === "list_maker" ? (
                          <Check className="h-4 w-4" />
                        ) : (
                          <Copy className="h-4 w-4" />
                        )}
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* 出价信息 */}
          {hasBid && (
            <div className="p-4 rounded-lg border border-blue-500/20 bg-blue-500/5">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold text-blue-500">当前最高出价</span>
                  <span className="text-xs bg-blue-500/20 text-blue-500 px-2 py-0.5 rounded">
                    可接受
                  </span>
                </div>
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">出价</span>
                  <span className="text-lg font-bold text-blue-500">
                    {formatPrice(itemDetail.bid_price)}
                  </span>
                </div>
                {itemDetail.bid_time && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">出价时间</span>
                    <span className="text-sm">{formatTime(itemDetail.bid_time)}</span>
                  </div>
                )}
                {itemDetail.bid_expire_time && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">过期时间</span>
                    <span className="text-sm">{formatTime(itemDetail.bid_expire_time)}</span>
                  </div>
                )}
                {itemDetail.bid_maker && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">出价者</span>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-mono">{formatAddress(itemDetail.bid_maker)}</span>
                      <button
                        onClick={() => copyToClipboard(itemDetail.bid_maker, "bid_maker")}
                        className="text-muted-foreground hover:text-foreground"
                      >
                        {copied === "bid_maker" ? (
                          <Check className="h-4 w-4" />
                        ) : (
                          <Copy className="h-4 w-4" />
                        )}
                      </button>
                    </div>
                  </div>
                )}
                {itemDetail.bid_size && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">出价数量</span>
                    <span className="text-sm">{itemDetail.bid_size}</span>
                  </div>
                )}
                {itemDetail.bid_unfilled !== undefined && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">未完成数量</span>
                    <span className="text-sm">{itemDetail.bid_unfilled}</span>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* 所有者信息 */}
          {itemDetail.owner_address && (
            <div className="p-4 rounded-lg border border-border bg-card">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">当前所有者</span>
                <div className="flex items-center gap-2">
                  <span className="text-sm font-mono">{formatAddress(itemDetail.owner_address)}</span>
                  <button
                    onClick={() => copyToClipboard(itemDetail.owner_address, "owner")}
                    className="text-muted-foreground hover:text-foreground"
                  >
                    {copied === "owner" ? (
                      <Check className="h-4 w-4" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* 其他信息 */}
          <div className="p-4 rounded-lg border border-border bg-card space-y-3">
            <h3 className="font-semibold mb-3">详细信息</h3>
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <div className="text-muted-foreground">链 ID</div>
                <div className="font-medium">{itemDetail.chain_id}</div>
              </div>
              <div>
                <div className="text-muted-foreground">集合地址</div>
                <div className="flex items-center gap-2">
                  <span className="font-mono text-xs">{formatAddress(itemDetail.collection_address)}</span>
                  <button
                    onClick={() => copyToClipboard(itemDetail.collection_address, "collection")}
                    className="text-muted-foreground hover:text-foreground"
                  >
                    {copied === "collection" ? (
                      <Check className="h-3 w-3" />
                    ) : (
                      <Copy className="h-3 w-3" />
                    )}
                  </button>
                </div>
              </div>
              {itemDetail.marketplace_id && (
                <div>
                  <div className="text-muted-foreground">市场 ID</div>
                  <div className="font-medium">{itemDetail.marketplace_id}</div>
                </div>
              )}
              {itemDetail.is_opensea_banned !== undefined && (
                <div>
                  <div className="text-muted-foreground">OpenSea 状态</div>
                  <div className="font-medium">
                    {itemDetail.is_opensea_banned ? "已封禁" : "正常"}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

