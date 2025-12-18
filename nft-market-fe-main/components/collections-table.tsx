import { ArrowDownIcon, ArrowUpIcon } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useRouter } from "next/navigation";
import { useGlobalState } from "@/hooks/useGlobalState";

type CollectionProps = {
  collections: any[];
  loading?: boolean;
};

// 生成随机钱包地址头像的函数
function getRandomAvatarUrl(): string {
  const services = [
    // Ethereum Blockies
    (addr: string) => `https://eth-blockies.herokuapp.com/${addr}.png`,
    // Dicebear Pixel Art
    (addr: string) => `https://api.dicebear.com/6.x/pixel-art/svg?seed=${addr}`,
    // Dicebear Avatars
    (addr: string) => `https://api.dicebear.com/6.x/avataaars/svg?seed=${addr}`,
  ];

  // 生成随机的钱包地址
  const randomAddr = '0x' + Array.from({length: 40}, () => 
    Math.floor(Math.random() * 16).toString(16)).join('');
  
  // 随机选择一个服务
  const randomService = services[Math.floor(Math.random() * services.length)];
  
  return randomService(randomAddr);
}

function hideAddress(address: string) {
  return address?.slice(0, 6) + '...' + address?.slice(-4);
}

export function CollectionsTable({ collections, loading }: CollectionProps) {
  const router = useRouter();
  // const { chain_id, collection_address } = useGlobalState();
  const { setState } = useGlobalState();
  return (
    <div className="px-6 py-4">
      <div className="relative">
        {/* 固定第一列的容器 */}
        <div className="absolute left-0 w-[80px] bg-background z-10">
          <Table>
            <TableHeader>
              <TableRow className="border-gray-800 h-[88px]">
                <TableHead className="w-[80px] text-foreground flex items-center justify-center">
                  <span>底价</span>
                  <ArrowUpIcon className="ml-1 h-4 w-4" />
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading
                ? Array.from({ length: 5 }).map((_, index) => (
                    <TableRow
                      key={`skeleton-${index}`}
                      className="border-gray-800 h-[88px]"
                    >
                      <TableCell className="w-[80px]">
                        <div className="w-[40px] h-[40px] rounded-md bg-gray-700 animate-pulse" />
                      </TableCell>
                    </TableRow>
                  ))
                : collections.map((collection) => (
                    <TableRow
                      key={collection.address}
                      className="border-gray-800 h-[88px] group hover:bg-gray-800 relative"
                    >
                      <TableCell className="w-[80px]">
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img
                          className="w-[40px] h-[40px] rounded-md"
                          src={collection.image_uri}
                          alt={collection.name}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
            </TableBody>
          </Table>
        </div>

        {/* 可滚动的其余列容器 */}
        <div className="overflow-x-auto ml-[80px]">
          <Table>
            <TableHeader>
              <TableRow className="border-gray-800 h-[88px] whitespace-nowrap">
                <TableHead className="w-[200px] text-foreground">
                  <></>
                </TableHead>
                <TableHead className="w-[120px] text-foreground">
                  最高出价
                </TableHead>
                <TableHead className="w-[120px] text-foreground">
                  1天内变化
                </TableHead>
                <TableHead className="w-[120px] text-foreground">
                  15天内成交量
                </TableHead>
                <TableHead className="w-[120px] text-foreground">
                  <div className="flex items-center">
                    1天内成交量
                    <ArrowDownIcon className="ml-1 h-4 w-4" />
                  </div>
                </TableHead>
                <TableHead className="w-[120px] text-foreground">
                  7天内成交量
                </TableHead>
                <TableHead className="w-[120px] text-foreground">
                  所有者
                </TableHead>
                <TableHead className="w-[120px] text-foreground">
                  供应量
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading
                ? Array.from({ length: 5 }).map((_, index) => (
                    <TableRow
                      key={`skeleton-${index}`}
                      className="h-[88px] border-gray-800"
                    >
                      <TableCell className="w-[200px]">
                        <div className="h-4 w-[120px] bg-gray-700 rounded animate-pulse" />
                      </TableCell>
                      <TableCell className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </TableCell>
                      <TableCell className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </TableCell>
                      <TableCell className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </TableCell>
                      <TableCell className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </TableCell>
                      <TableCell className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </TableCell>
                      <TableCell className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </TableCell>
                      <TableCell className="w-[120px]">
                        <div className="h-4 w-16 bg-gray-700 rounded animate-pulse" />
                      </TableCell>
                    </TableRow>
                  ))
                : collections.map((collection) => (
                    <TableRow
                      key={collection.address}
                      className="h-[88px] border-gray-800 group hover:bg-gray-800 cursor-pointer relative"
                      onClick={() => {
                        setState({
                          chain_id: collection.chain_id,
                          collection_address: collection.address,
                        });
                        router.push(
                          `/collections/${encodeURIComponent(collection.address)}`
                        );
                      }}
                    >
                      <TableCell className="w-[200px]">
                        {collection.name}
                      </TableCell>
                      <TableCell className="w-[120px]">
                        {collection.floor_price}
                      </TableCell>
                      <TableCell className="w-[120px]">
                        {collection.floor_price_change}
                      </TableCell>
                      <TableCell className="w-[120px]">
                        {collection.item_sold}
                      </TableCell>
                      <TableCell className="w-[120px]">
                        {collection.item_sold}
                      </TableCell>
                      <TableCell className="w-[120px]">
                        {collection.item_sold}
                      </TableCell>
                      <TableCell className="w-[120px]">
                        <div className="flex items-center">
                          <img src={getRandomAvatarUrl()} alt={collection.name} className="w-[40px] h-[40px] rounded-md" />
                          {hideAddress(collection.owner)}
                        </div>
                      </TableCell>
                      <TableCell className="w-[120px]">
                        {collection.item_num}
                      </TableCell>
                    </TableRow>
                  ))}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}
