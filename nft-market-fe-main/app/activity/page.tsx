// @ts-nocheck

"use client";


import { useEffect, useState } from "react";
import { Switch } from "@headlessui/react";
import { ChartComponent } from "./components/chart";
import { RiMenuLine, RiBarChartLine } from "@remixicon/react";
import activityApi from "@/api/activity";

export default function TradingDashboard() {
  const [showRelative, setShowRelative] = useState(true);
  const [showThirtyDays, setShowThirtyDays] = useState(true);
  const [showGrid, setShowGrid] = useState(true);

  const [transactionData, setTransactionData] = useState([]);

  const [collections, setCollections] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  async function getActivity() {
    setLoading(true);
    const res = await activityApi.GetActivity({
      filter_ids: [],
      event_types: [],
      user_addresses: [],
      collection_addresses: [],
      page: 1,
      page_size: 50,
    });
    // @ts-ignore
    setCollections(sortCollections(res.result));
    // @ts-ignore
    setTransactionData(res.result);
    setLoading(false);
  }

  function sortCollections(activities: any[]) {
    const collectionMap: Record<
      string,
      { count: number; image_uri: string; name: string; address: string }
    > = {};
    
    activities.forEach((item) => {
    // console.log(item, '===');

      if (!collectionMap[item.collection_address]) {
        collectionMap[item.collection_address] = {
          image_uri: item.collection_image_uri,
          name: item.collection_name,
          count: 1,
          address: item.collection_address,
        };
      } else {
        collectionMap[item.collection_address].count++;
      }

      item.rarity = getRandomRarity();
    });
    // setCollections(Object.values(collectionMap));

    // console.log(Object.values(collectionMap));
    
    return Object.values(collectionMap);
    // return activities.sort((a, b) => a.rarity - b.rarity);
  }

  useEffect(() => {
    getActivity();
  }, []);

  return (
    <div className="min-h-screen bg-black text-white">
      <div className="flex">
        {/* Left Sidebar */}
        <div className="w-64 border-r border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-6">
            <RiMenuLine className="h-5 w-5" />
            <span>种类</span>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Switch
                  checked={true}
                  onChange={() => {}}
                  className={`relative inline-flex h-5 w-9 items-center rounded-full bg-[#b599fd]`}
                >
                  <span className="translate-x-5 inline-block h-3 w-3 transform rounded-full bg-white transition" />
                </Switch>
                <span>销售</span>
              </div>
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Switch
                  checked={true}
                  onChange={() => {}}
                  className={`relative inline-flex h-5 w-9 items-center rounded-full bg-[#b599fd]`}
                >
                  <span className="translate-x-5 inline-block h-3 w-3 transform rounded-full bg-white transition" />
                </Switch>
                <span>挂单</span>
              </div>
            </div>
          </div>

          <div className="mt-8">
            <div className="flex items-center gap-2 mb-6">
              <RiBarChartLine className="h-5 w-5" />
              <span>系列</span>
              <span className="text-gray-500 text-sm ml-auto">重置</span>
            </div>

            <div className="relative">
              <input
                type="text"
                placeholder="搜索"
                className="w-full bg-gray-900 rounded-lg px-4 py-2 text-sm"
              />
            </div>

            <div className="mt-4 space-y-3">
              {collections.map((item) => (
                <div
                  key={item.address}
                  className="flex items-center justify-between"
                >
                  <div className="flex items-center gap-2">
                    <img
                      src={item.image_uri || "/placeholder.svg"}
                      alt=""
                      className="w-6 h-6 rounded-full"
                    />
                    <span className="text-sm">{item.name}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm">{item.price}</span>
                    <span
                      className={`text-sm ${
                        item.change >= 0 ? "text-green-500" : "text-red-500"
                      }`}
                    >
                      {item.change > 0 ? "+" : ""}
                      {item.change || 0}%
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Main Content */}
        <div className="flex-1 p-4">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-2">
              <RiBarChartLine className="h-5 w-5" />
              <span>活动</span>
            </div>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <span>相关</span>
                <Switch
                  checked={showRelative}
                  onChange={setShowRelative}
                  className={`relative inline-flex h-5 w-9 items-center rounded-full ${
                    showRelative ? "bg-[#b599fd]" : "bg-gray-700"
                  }`}
                >
                  <span
                    className={`${
                      showRelative ? "translate-x-5" : "translate-x-1"
                    } inline-block h-3 w-3 transform rounded-full bg-white transition`}
                  />
                </Switch>
              </div>
              <div className="flex items-center gap-2">
                <span>30天</span>
                <Switch
                  checked={showThirtyDays}
                  onChange={setShowThirtyDays}
                  className={`relative inline-flex h-5 w-9 items-center rounded-full ${
                    showThirtyDays ? "bg-[#b599fd]" : "bg-gray-700"
                  }`}
                >
                  <span
                    className={`${
                      showThirtyDays ? "translate-x-5" : "translate-x-1"
                    } inline-block h-3 w-3 transform rounded-full bg-white transition`}
                  />
                </Switch>
              </div>
              <div className="flex items-center gap-2">
                <span>显示图表</span>
                <Switch
                  checked={showGrid}
                  onChange={setShowGrid}
                  className={`relative inline-flex h-5 w-9 items-center rounded-full ${
                    showGrid ? "bg-[#b599fd]" : "bg-gray-700"
                  }`}
                >
                  <span
                    className={`${
                      showGrid ? "translate-x-5" : "translate-x-1"
                    } inline-block h-3 w-3 transform rounded-full bg-white transition`}
                  />
                </Switch>
              </div>
            </div>
          </div>

          {/* <div className="h-[400px] mb-6">
            <ChartComponent />
          </div> */}

          <div className="bg-gray-900 rounded-lg">
            <table className="w-full">
              <thead>
                <tr className="text-gray-500 text-sm">
                  <th className="text-left p-4">操作</th>
                  <th className="text-left p-4">物品</th>
                  <th className="text-left p-4">稀有度</th>
                  <th className="text-right p-4">价格</th>
                  <th className="text-center p-4">最高出价</th>
                  <th className="text-left p-4">从</th>
                  <th className="text-left p-4">至</th>
                  <th className="text-right p-4">时间</th>
                </tr>
              </thead>
              <tbody>

              {loading && Array.from({ length: 5 }).map((_, index) => (
                    <tr
                      key={`skeleton-${index}`}
                      className="h-[88px] border-gray-800"
                    >
                      <td className="w-[200px]">
                        <div className="h-4 w-[120px] bg-gray-700 rounded animate-pulse" />
                      </td>
                      <td className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </td>
                      <td className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </td>
                      <td className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </td>
                      <td className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </td>
                      <td className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </td>
                      <td className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </td>
                      <td className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </td>
                    </tr>
                  ))}
                {transactionData.map((item, index) => (
                  <tr key={index} className="border-t border-gray-800">
                    <td className="p-4">
                      {item.type === "sell" ? "销售" : "挂单"}
                    </td>
                    <td className="p-4">
                      <div className="flex items-center gap-2">
                        <img
                          src={item.image_url || getRandomNftImage()}
                          alt=""
                          className="w-8 h-8 rounded-lg"
                        />
                        <div>
                          <div>{item.collection_name}</div>
                          <div className="text-sm text-gray-500">
                            {item.item_name}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className={`p-4 ${item.rarity?.color}`}>{item.rarity?.name}</td>
                    <td className="p-4 text-right">{weiToEth(item.price)}</td>
                    <td className="p-4 text-center">
                      {item.highestBid ? weiToEth(item.highestBid) : '-'}</td>
                    <td className="p-4">{item.from || '-'}</td>
                    <td className="p-4">{item.to || '-'}</td>
                    <td className="p-4 text-right">
                      {formatTime(item.event_time)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}

type Rarity = {
  name: string;
  color: string;
  probability: number;
};

const rarityLevels: Rarity[] = [
  { name: "Common", color: "text-gray-400", probability: 50 }, // 50% 概率
  { name: "Uncommon", color: "text-green-400", probability: 25 }, // 25% 概率
  { name: "Rare", color: "text-blue-400", probability: 15 }, // 15% 概率
  { name: "Epic", color: "text-purple-400", probability: 7 }, // 7% 概率
  { name: "Legendary", color: "text-yellow-400", probability: 2.5 }, // 2.5% 概率
  { name: "Mythic", color: "text-red-400", probability: 0.5 }, // 0.5% 概率
];

function getRandomRarity(): Rarity {
  const random = Math.random() * 100; // 生成 0-100 的随机数
  let probabilitySum = 0;

  for (const rarity of rarityLevels) {
    probabilitySum += rarity.probability;
    if (random <= probabilitySum) {
      return rarity;
    }
  }

  return rarityLevels[0]; // 默认返回 Common
}

const nftImages = [
  "https://i.seadn.io/gae/Ju9CkWtV-1Okvf45wo8UctR-M9He2PjILP0oOvxE89AyiPPGtrR3gysu1Zgy0hjd2xKIgjJJtWIc0ybj4Vd7wv8t3pxDGHoJBzDB?w=500&auto=format",
  "https://i.seadn.io/gae/H8jOCJuQokNqGBpkBN5wk1oZwO7LM8bNnrHCaekV2nKjnCqw6UB5oaH8XyNeBDj6bA_n1mjejzhFQUP3O1NfjFLHr3FOaeHcTOOT?w=500&auto=format",
  "https://i.seadn.io/gcs/files/c6cb0b1d6f2ab61c0efacf00e62e2230.jpg?w=500&auto=format",
  "https://i.seadn.io/gcs/files/a8a2c681f0437294a88d4fd4cd161a0d.png?w=500&auto=format",
  "https://i.seadn.io/gae/BdxvLseXcfl57BiuQcQYdJ64v-aI8din7WPk0Pgo3qQFhAUH-B6i-dCqqc_mCkRIzULmwzwecnohLhrcH8A9mpWIZqA7ygc52Sr81hE?w=500&auto=format",
  "https://i.seadn.io/gae/7B0qai02OdHA8P_EOVK672qUliyjQdQDGNrACxs7WnTgZAkJa_wWURnIFKeOh5VTf8cfTqW3wQpozGedaC9mteKphEOtztls02RlWQ?w=500&auto=format",
  "https://i.seadn.io/gae/Qe-nxjnG1ZNrxQvVvR4aI5SyLuqT4oKw7XJYVLBJNJzNOScPbpClW6e1wm2sUYPzQWBc2WBCFzGHGFsj1fHxZTBs3VVJ2u-0YnQF?w=500&auto=format",
  "https://i.seadn.io/gae/yIm-M5-BpSDdTEIJRt5D6xphizhIdozXjqSITgK4phWq7MmAU3qE7Nw7POGCiPGyhtJ3ZFP8iJ29TFl-RLcGBWX5qI4-ZcnCPcsY4zI?w=500&auto=format",
  "https://i.seadn.io/gae/PHxWz47uWRKHGnUBk-CkWZdUiE6kzX_sgvQN1YKBn45qZbI3Dj3RXwwh-9Xb2xlwqkEUF9_o-ySm_uqEICe0v-nfE3mQB3skjENYXA?w=500&auto=format",
  "https://i.seadn.io/gae/6X867ZmCsuYcjHpx-nmNkXeHaDFd2m-EDEEkExVLKETphkfcrpRJOyzFxRQlc-29J0e-9mB9uDGze0O9yracSA9ibnQm2sIq5i2Yuw?w=500&auto=format",
  "https://i.seadn.io/gae/d784iHHbqQFVH1XYD6HoT4u3y_Fsu_9FZUltWjnOzoYv7qqB5dLUqpGyHBd8Gq3h4mykK5PtkYzdGGoryWJaiFzGyx0_cWbwwE_W?w=500&auto=format",
  "https://i.seadn.io/gae/8g0poMCQ5J9SZHMsQ3VD3C_8zoXzMJThmm0AwA11RuKGS4lqXPzrZlqKVUZ5PzhWS_GqLTOFoZLgXq6XTVQUqkz1XPq6AmwLd9Wz?w=500&auto=format",
  "https://i.seadn.io/gae/LIov33kogXOK4XZd2ESj29sqm_Hww5JSdO7AFn5wjt8xgnJJ0UpNV9yITqxra3s_LMEW1AnnrgOVB_hDpjJRA1uF4skI5Sdi_9rULi8?w=500&auto=format",
  "https://i.seadn.io/gae/obxP_zMXHWA8qZK4GkdD3HGQGLXJGXfs6GPbHCXxeqQWcjVEfqMoaix7dY7uUc8yHGQA4iBIKZpsTIxv0qEQqtYYxHdBCHe6_qhY?w=500&auto=format",
  "https://i.seadn.io/gae/9EPP0IGrMUgR8FgFd-RTo4epMm55JUEOVaUvGY9CKQM9J47Ni6OxOTbQKeDwS_8BXo5IKL4H_Q0tZ8pK2UsE1eEpyVB6_T_GvgD9=w600",
  "https://i.seadn.io/gae/KbTJGEK-dRFtvh56IY6AaWFqJJylvCjYqjMJKF8ymx4iWzU8lYS3nMXoHHHVE1Zqb8QWGRJz3qB9uPcBDJMv-vqpUs7GS-qjRANj?w=500&auto=format",
  "https://i.seadn.io/gae/UA7blhz93Jk8BwqDf6EX6Rc8teS_T5KGGXz9p8XkI6u8oQhh4WkHrPKqQqQd7E3nPTvZ-_uFX_WTLKgiDHd6MuuUQb_J2OHE4Iqn?w=500&auto=format",
  "https://i.seadn.io/gae/6ev2GVLx3BDJq89890xrLuoS6kNT7H0C_cC5-Tg4uBM0PSvKGkqIxF0K1E0fS9kJey9B5AGFnQwU3e9HzJ75f-Z0hpRVL-O-mQlZ?w=500&auto=format",
  "https://i.seadn.io/gae/ZWEV7BBCrHlVe8nOh0ySo5uZQxqwMTZ7_GQu0cLrGQWcHJTGx2cEyZyQHf7j_KhxZMsBxnKhc_UDmFV-VUG0bBLM3rqYTGddGhFS?w=500&auto=format",
  "https://i.seadn.io/gae/0cOqWoYA7xL9CkUjGlxsjreSYBdrUBE0c6EO1COG4XE8UeP-Z30ckqUNiL872zHQHQU5MUNMNhvDpyXIP17hRUS7ZJT31O5ieZKXs1Y?w=500&auto=format",
];

function getRandomNftImage(): string {
  const randomIndex = Math.floor(Math.random() * nftImages.length);
  return nftImages[randomIndex];
}

function weiToEth(wei: string | number): string {
  const value = typeof wei === "string" ? wei : wei.toString();
  const ethValue = Number(value) / 1e16;
  return ethValue.toFixed(2) + "ETH";
}

function formatTime(seconds: number): string {
  //   const minutes = Math.floor(seconds / 60);
  //   const hours = Math.floor(minutes / 60);
  //   const days = Math.floor(hours / 24);

  const date = new Date(seconds * 1000);

  const diff = Date.now() - date.getTime();
  const days = Math.floor(diff / (1000 * 60 * 60 * 24));
  const hours = Math.floor(diff / (1000 * 60 * 60));
  const minutes = Math.floor(diff / (1000 * 60));
  //   const seconds = Math.floor(diff / 1000);

  if (days > 0) {
    return `${days}天前`;
  } else if (hours > 0) {
    return `${hours}小时前`;
  } else if (minutes > 0) {
    return `${minutes}分钟前`;
  } else {
    return `${seconds}秒前`;
  }
}
