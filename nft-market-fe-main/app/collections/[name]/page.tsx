"use client";

import { CollectionView } from "./components/collection-view";
import { CollectionHeader } from "./components/collection-header";
import collectionApi from "@/api/collections";
import { useEffect, useState } from "react";
import { useGlobalState } from "@/hooks/useGlobalState";

export default function Page() {
  const { state, setState } = useGlobalState();

  const [collectionItems, setCollectionItems] = useState<any[]>([]);
  const [collectionParams, setCollectionParams] = useState<any>({
    sort: 1,
    status: [],
    markets: [],
    token_id: "",
    user_address: state.user_address,
    chain_id: state.chain_id,
    page: 1,
    page_size: 20,
    count: -1,
  });
  useEffect(() => {
    getCollectionDetail();
    getCollectionItems();
  }, []);

  async function getCollectionDetail() {
    // 从 path 中读取address
    const address = window.location.pathname.split("/").pop() || "";
    const res = await collectionApi.GetCollectionDetail({
      address,
      chain_id: state.chain_id,
    });
    // console.log(res, '=res')

    setState({
      // @ts-ignore
      collection: res.result,
    });
  }

  async function getCollectionItems() {
    const res = await collectionApi.GetCollectionItems({
      address: state.collection_address,
      filters: {
        sort: 1,
        status: [],
        markets: [],
        // token_id: '0',
        user_address: state.user_address,
        chain_id: state.chain_id,
        page: 1,
        page_size: 20,
      },
    });
    console.log(res, "=res");
    setCollectionItems(res?.result || []);
    setCollectionParams({
      ...collectionParams,
      count: res?.count,
    });
  }

  return (
    <main className="min-h-screen bg-background text-foreground">
      <CollectionHeader />
      <CollectionView items={collectionItems} chainId={state.chain_id} />
    </main>
  );
}
