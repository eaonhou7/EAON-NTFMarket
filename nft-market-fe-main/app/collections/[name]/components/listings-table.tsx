import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Button } from "@/components/ui/button"

export function ListingsTable() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm">
            稀有度
          </Button>
          <Button variant="outline" size="sm">
            立即购买
          </Button>
          <Button variant="outline" size="sm">
            最新成交
          </Button>
          <Button variant="outline" size="sm">
            最高出价
          </Button>
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>已挂单</TableHead>
            <TableHead>稀有度</TableHead>
            <TableHead>立即购买</TableHead>
            <TableHead>最新成交</TableHead>
            <TableHead>最高出价</TableHead>
            <TableHead>持有者数量</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow>
            <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
              未找到数据。
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  )
}

