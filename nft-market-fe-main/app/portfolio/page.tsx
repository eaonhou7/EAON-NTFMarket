"use client";

import { Sidebar } from "./components/sidebar";
import { MainContent } from "./components/main-content";
import { useAccount } from "wagmi";
import { useEffect, useState } from "react";
import activityApi from "@/api/activity";

let myListOrdersPage = 1;
const myListOrdersPageSize = 10;
export default function Home() {

  const { address: owner, chainId } = useAccount();

  const [myListOrdersCount, setMyListOrdersCount] = useState<number>(0);
  const [myListOrders, setMyListOrders] = useState<any[]>([]);

  const loadMyListOrders = async () => {
    if (!owner || !chainId) return;
    // 已经全部加载完了
    if (myListOrdersPage * myListOrdersPageSize >= myListOrdersCount && myListOrdersCount > 0) return;
  
    const params = {
      filter_ids: [chainId],
      user_addresses: [owner],
      event_types: ['list'],
      page: myListOrdersPage,
      page_size: myListOrdersPageSize,
    }
    // @ts-ignore
    const {count, result} = await activityApi.GetActivity(params);
    setMyListOrdersCount(count + myListOrdersCount);
    console.log();
    
    setMyListOrders([...myListOrders, ...result]);
    myListOrdersPage++;
  }

  useEffect(() => {
    loadMyListOrders();
  }, [owner]);

  return (
    <div className="flex flex-col min-h-screen">
      {/* <Header /> */}
      {/* <WarningBanner /> */}
      <div className="flex flex-1">
        <Sidebar />
        <MainContent myListOrders={myListOrders}/>
      </div>
    </div>
  );
}
