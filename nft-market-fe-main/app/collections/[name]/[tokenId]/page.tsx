"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import collectionApi from "@/api/collections";
import { useGlobalState } from "@/hooks/useGlobalState";
import { ItemDetailView } from "./components/item-detail-view";
import { CollectionHeader } from "../components/collection-header";

export default function ItemDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { state } = useGlobalState();
  
  const [itemDetail, setItemDetail] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const collectionAddress = params.name as string;
  const tokenId = params.tokenId as string;

  const fetchItemDetail = useCallback(async () => {
    if (!collectionAddress || !tokenId || !state.chain_id) {
      return;
    }
    try {
      setLoading(true);
      setError(null);
      const res: any = await collectionApi.getCollectionItemDetail(
        collectionAddress,
        tokenId,
        state.chain_id
      );
      setItemDetail(res?.result || null);
    } catch (err: any) {
      console.error("Failed to fetch item detail:", err);
      setError(err?.message || "获取物品详情失败");
    } finally {
      setLoading(false);
    }
  }, [collectionAddress, tokenId, state.chain_id]);

  useEffect(() => {
    fetchItemDetail();
  }, [fetchItemDetail]);

  if (loading) {
    return (
      <main className="min-h-screen bg-background text-foreground">
        <CollectionHeader />
        <div className="container mx-auto px-4 py-16">
          <div className="text-center">
            <div className="text-6xl mb-4">⏳</div>
            <h3 className="text-xl font-semibold mb-2">加载中...</h3>
            <p className="text-muted-foreground">正在获取物品详情</p>
          </div>
        </div>
      </main>
    );
  }

  if (error || !itemDetail) {
    return (
      <main className="min-h-screen bg-background text-foreground">
        <CollectionHeader />
        <div className="container mx-auto px-4 py-16">
          <div className="text-center">
            <div className="text-6xl mb-4">❌</div>
            <h3 className="text-xl font-semibold mb-2">加载失败</h3>
            <p className="text-muted-foreground mb-4">
              {error || "未找到该物品"}
            </p>
            <button
              onClick={() => router.back()}
              className="text-primary hover:underline"
            >
              返回上一页
            </button>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-background text-foreground">
      <CollectionHeader />
      <ItemDetailView itemDetail={itemDetail} chainId={state.chain_id} />
    </main>
  );
}

