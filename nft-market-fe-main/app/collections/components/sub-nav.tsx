'use client'

import { TrendingUp, Star, Target, Grid } from 'lucide-react'
import { useContext } from 'react'
import { useRouter } from 'next/navigation'
import { CollectionsContext } from '../hooks/context'

export function SubNav() {
    const { activeTab, setActiveTab } = useContext(CollectionsContext);
    const router = useRouter()

    function handleTabClick(tab: string) {
        router.push(`#${tab}`)
        setActiveTab(tab)
    }

    return (
    <div className="border-b border-gray-800 px-2 sm:px-6">
      <div className="flex flex-col sm:flex-row items-start sm:items-center gap-2 sm:space-x-8 py-2 text-sm">
        {/* <a onClick={() => handleTabClick('collections')} 
           className={`flex w-full sm:w-auto items-center space-x-2 p-2 sm:p-3 text-gray-400 rounded-[8px] border-[1px] ${
             activeTab === 'collections' ? 'border-primary text-primary' : 'border-transparent'
           }`}>
          <Grid className="w-4 h-4" />
          <span>COLLECTIONS</span>
        </a> */}
        <a onClick={() => handleTabClick('trending')} 
           className={`flex w-full sm:w-auto items-center space-x-2 p-2 sm:p-3 text-gray-400 rounded-[8px] hover:text-white border-[1px] ${
             activeTab === 'trending' ? 'border-primary text-primary' : 'border-transparent'
           }`}>
          <TrendingUp className="w-4 h-4" />
          <span>热门</span>
        </a>
        {/* <a onClick={() => handleTabClick('favorites')} 
           className={`flex w-full sm:w-auto items-center space-x-2 p-2 sm:p-3 text-gray-400 rounded-[8px] hover:text-white border-[1px] ${
             activeTab === 'favorites' ? 'border-primary text-primary' : 'border-transparent'
           }`}>
          <Star className="w-4 h-4" />
          <span>我收藏的</span>
        </a> */}
        {/* <a onClick={() => handleTabClick('points')} 
           className={`flex w-full sm:w-auto items-center space-x-2 p-2 sm:p-3 text-gray-400 rounded-[8px] hover:text-white border-[1px] ${
             activeTab === 'points' ? 'border-primary text-primary' : 'border-transparent'
           }`}>
          <Target className="w-4 h-4" />
          <span>积分</span>
        </a> */}
      </div>
    </div>
  )
}

