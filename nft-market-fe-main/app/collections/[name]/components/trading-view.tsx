import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Switch } from "@/components/ui/switch"
import { Settings2 } from "lucide-react"

export function TradingView() {
  return (
    <div className="rounded-lg border border-border p-4 space-y-4">
      <div className="flex items-center justify-between">
        <Tabs defaultValue="depth">
          <TabsList>
            <TabsTrigger value="depth">深度</TabsTrigger>
            <TabsTrigger value="volume">销量</TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Switch id="sweep" />
            <label htmlFor="sweep" className="text-sm">
              优化多件购买 (SWEEP) 功能
            </label>
          </div>
          <Button variant="outline" size="icon">
            <Settings2 className="h-4 w-4" />
          </Button>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm">
            1天
          </Button>
          <Button variant="outline" size="sm">
            1周
          </Button>
          <Button variant="outline" size="sm">
            1个月
          </Button>
        </div>
      </div>

      <div className="h-[200px] bg-muted rounded-lg" />

      <div className="flex items-center justify-between text-sm text-muted-foreground">
        <div className="flex items-center gap-4">
          <div>18时</div>
          <div>23时</div>
          <div>4时</div>
          <div>9时</div>
          <div>14时</div>
        </div>
      </div>
    </div>
  )
}

