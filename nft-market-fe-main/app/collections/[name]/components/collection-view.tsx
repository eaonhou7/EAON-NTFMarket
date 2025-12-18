"use client"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ItemsGrid } from "./items-grid"
import { TradingView } from "./trading-view"
import { RecentActivity } from "./recent-activity"

interface CollectionViewProps {
  items?: any[]
  chainId?: string | number
}

export function CollectionView({ items = [], chainId }: CollectionViewProps) {
  return (
    <div className="container mx-auto px-4 py-6">
      <Tabs defaultIndex={0} className="space-y-4">
        {/* <TabsList>
          <TabsTrigger>物品</TabsTrigger>
          <TabsTrigger>活动</TabsTrigger>
          <TabsTrigger>分析</TabsTrigger>
        </TabsList> */}
        <TabsContent>
          <div className="grid grid-cols-[1fr_300px] gap-6">
            <div className="min-h-[400px]">
              <ItemsGrid items={items} chainId={chainId} />
            </div>
            <RecentActivity />
          </div>
          <TradingView />
        </TabsContent>
        <TabsContent>{/* 活动内容 */}</TabsContent>
        <TabsContent>{/* 分析内容 */}</TabsContent>
      </Tabs>
    </div>
  )
}

