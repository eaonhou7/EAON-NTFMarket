"use client"

import { formatUnits } from "ethers"
import { useState } from "react"
import { ExternalLink } from "lucide-react"
import { useRouter, usePathname } from "next/navigation"
import { formatAddress, getNftExplorerUrl } from "@/lib/utils"
import { Button } from "@/components/ui/button"

// ETH 使用 18 位小数
const ETH_DECIMALS = 18

interface Item {
  name: string
  image_uri: string
  token_id: string
  collection_address: string
  owner_address: string
  list_price: string
  bid_price: string
  last_sell_price: string
  list_order_id: string
  bid_order_id: string
  owner_owned_amount: number
  rarity_rank?: number
  [key: string]: any
}

interface ItemsGridProps {
  items: Item[]
  chainId?: string | number
}

export function ItemsGrid({ items, chainId = 11155111 }: ItemsGridProps) {
  const router = useRouter()
  const pathname = usePathname()
  const [imageErrors, setImageErrors] = useState<Set<string>>(new Set())

  const formatPrice = (price: string, decimals: number = ETH_DECIMALS): string => {
    if (!price || price === "0") return "-"
    try {
      const ethValue = formatUnits(price, decimals)
      const numValue = parseFloat(ethValue)
      if (numValue === 0) return "-"
      // 如果值很大，显示更多小数位；如果值很小，显示更多小数位
      if (numValue >= 1) {
        return `${numValue.toFixed(4)} ETH`
      } else if (numValue >= 0.01) {
        return `${numValue.toFixed(6)} ETH`
      } else {
        return `${numValue.toFixed(8)} ETH`
      }
    } catch {
      return "-"
    }
  }

  const handleImageError = (itemKey: string) => {
    setImageErrors((prev) => new Set(prev).add(itemKey))
  }

  if (items.length === 0) {
    return (
      <div className="text-center py-16">
        <div className="text-6xl mb-4">🎨</div>
        <h3 className="text-xl font-semibold text-foreground mb-2">暂无物品</h3>
        <p className="text-muted-foreground">该集合中还没有物品</p>
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
      {items.map((item) => {
        const itemKey = `${item.collection_address}-${item.token_id}`
        const hasImageError = imageErrors.has(itemKey)

        const handleItemClick = () => {
          router.push(`${pathname}/${item.token_id}`)
        }

        return (
          <div
            key={itemKey}
            onClick={handleItemClick}
            className="group relative rounded-xl border border-border bg-card hover:border-primary/50 transition-all duration-200 overflow-hidden cursor-pointer"
          >
            {/* NFT 图片 */}
            <div className="aspect-square w-full relative bg-muted overflow-hidden group/image">
              {!hasImageError && item.image_uri ? (
                <img
                  src={item.image_uri}
                  alt={item.name || `Token #${item.token_id}`}
                  className="w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
                  onError={() => handleImageError(itemKey)}
                />
              ) : (
                <div className="absolute inset-0 bg-muted flex items-center justify-center">
                  <div className="text-center">
                    <div className="text-4xl mb-2">🖼️</div>
                    <div className="text-xs text-muted-foreground">图片加载失败</div>
                  </div>
                </div>
              )}
              {/* 区块链浏览器链接按钮 */}
              <div className="absolute top-2 right-2 opacity-0 group-hover/image:opacity-100 transition-opacity">
                <Button
                  variant="secondary"
                  size="sm"
                  className="h-8 w-8 p-0 bg-background/80 backdrop-blur-sm hover:bg-background/90"
                  onClick={(e) => {
                    e.stopPropagation()
                    const url = getNftExplorerUrl(chainId, item.collection_address, item.token_id)
                    window.open(url, '_blank', 'noopener,noreferrer')
                  }}
                  title="在区块链浏览器中查看"
                >
                  <ExternalLink className="h-4 w-4" />
                </Button>
              </div>
            </div>

            {/* NFT 信息 */}
            <div className="p-4 space-y-2">
              <div>
                <h3 className="font-semibold text-foreground truncate">
                  {item.name || `Token #${item.token_id}`}
                </h3>
                <div className="flex items-center gap-2 mt-1 flex-wrap">
                  <span className="text-xs text-muted-foreground">#{item.token_id}</span>
                  {item.rarity_rank && item.rarity_rank > 0 && (
                    <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded">
                      稀有度 #{item.rarity_rank}
                    </span>
                  )}
                  {item.owner_owned_amount > 1 && (
                    <span className="text-xs bg-secondary/50 text-secondary-foreground px-2 py-0.5 rounded">
                      {item.owner_owned_amount} 个
                    </span>
                  )}
                </div>
              </div>

              {/* 状态标签 */}
              <div className="flex items-center gap-2 flex-wrap">
                {item.list_order_id && item.list_order_id !== "" && (
                  <span className="text-xs bg-green-500/10 text-green-500 px-2 py-0.5 rounded">
                    已挂单
                  </span>
                )}
                {item.bid_order_id && item.bid_order_id !== "" && (
                  <span className="text-xs bg-blue-500/10 text-blue-500 px-2 py-0.5 rounded">
                    有出价
                  </span>
                )}
              </div>

              {/* 价格信息 */}
              <div className="space-y-1.5 pt-2 border-t border-border">
                {item.list_price && item.list_price !== "0" && (
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">挂单价格</span>
                    <span className="text-sm font-semibold text-foreground">
                      {formatPrice(item.list_price)}
                    </span>
                  </div>
                )}
                {item.bid_price && item.bid_price !== "0" && (
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">最高出价</span>
                    <span className="text-sm text-blue-400 font-medium">
                      {formatPrice(item.bid_price)}
                    </span>
                  </div>
                )}
                {item.last_sell_price && item.last_sell_price !== "0" && (
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">最后成交</span>
                    <span className="text-sm text-foreground">
                      {formatPrice(item.last_sell_price)}
                    </span>
                  </div>
                )}
                {(!item.list_price || item.list_price === "0") &&
                  (!item.bid_price || item.bid_price === "0") &&
                  (!item.last_sell_price || item.last_sell_price === "0") && (
                    <div className="text-xs text-muted-foreground">暂无价格信息</div>
                  )}
              </div>

              {/* 持有者信息 */}
              {item.owner_address && (
                <div className="pt-2 border-t border-border">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">持有者</span>
                    <span className="text-xs text-foreground font-mono">
                      {formatAddress(item.owner_address)}
                    </span>
                  </div>
                </div>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}

