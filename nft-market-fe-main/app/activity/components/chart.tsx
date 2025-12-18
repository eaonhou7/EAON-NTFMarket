"use client"

export function ChartComponent() {
  return (
    <div className="w-full h-full bg-[#111] rounded-lg p-4">
      <div className="flex justify-between mb-4">
        <div className="space-y-1">
          <div className="text-2xl">58.22%</div>
          <div className="text-sm text-gray-500">最高涨幅</div>
        </div>
        <div className="space-y-1">
          <div className="text-2xl">-23.95%</div>
          <div className="text-sm text-gray-500">最低跌幅</div>
        </div>
      </div>

      {/* Placeholder for actual chart implementation */}
      <div className="w-full h-[300px] relative">
        <div className="absolute inset-0 flex items-center justify-center text-gray-500">
          Chart area - Would implement with a charting library
        </div>
      </div>
    </div>
  )
}

