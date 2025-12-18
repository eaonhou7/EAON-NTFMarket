"use client";
import { CollectionsTable } from "@/components/collections-table";
import collectionApi from "@/api/collections";
import portfolioApi from "@/api/portfolio";
import { useEffect, useState } from "react";
import { useGlobalState } from "@/hooks/useGlobalState";
import { useAccount } from "wagmi";

type CollectionsProps = {
  type: string;
};

export default function Collections({ type }: CollectionsProps) {
  const [collections, setCollections] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const { state, setState } = useGlobalState();
  const { address } = useAccount();

  useEffect(() => {
    if (address) {
      setState({ walletAddress: address as string });
    }
  }, [address]);

  async function fetchCollections() {
    setLoading(true);
     
    // if (type == "trending") {
      const res = await collectionApi.GetCollections({
        limit: 10,
        range: "1d",
      });
      // @ts-ignore
      setCollections(res?.result || []);
      setLoading(false);
    // } else {
    //   const res = await portfolioApi.GetPortfolio({
    //     filters: {
    //       user_addresses: [state.walletAddress],
    //     },
    //   });
    //   // @ts-ignore
    //   setCollections(res?.result?.collection_info || []);
    //   setLoading(false);
    // }
   
  }

  useEffect(() => {
    fetchCollections();
    // if (type != "trending") {
    //   fetchMyCollections();
    // }
  }, []);

  // async function fetchMyCollections() {
  //   setLoading(true);
     
  //   const res = await portfolioApi.GetPortfolio({
  //     filters: {
  //       user_addresses: [state.walletAddress],
  //     },
  //   });
  //   // @ts-ignore
  //   setCollections(res?.result?.collection_info || []);
  //   setLoading(false);
  // }

  return (
    <div>
      <CollectionsTable collections={collections} loading={loading} />
    </div>
  );
}
