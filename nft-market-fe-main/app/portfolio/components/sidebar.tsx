"use client"

import { RadioGroup } from "@headlessui/react"
import { useAccount } from 'wagmi'
import { Search, Eye } from "lucide-react"

export function Sidebar() {
  const { address } = useAccount();
  return (
    <div className="w-72 border-r border-gray-800 p-4">
      <div className="flex items-center space-x-2 mb-6">
        <div className="h-10 w-10 rounded-full bg-[rgb(95,198,212)] flex items-center justify-center">
        🐒
        </div>
        <div className="text-sm">{formatAddress(address)}</div>
      </div>

      <div className="mb-6">
        <RadioGroup>
          <RadioGroup.Option value="owned">
            {({ checked }) => (
              <div
                className={`${checked ? "bg-[#8e67e9] text-white" : "bg-gray-900 text-gray-300"} relative flex cursor-pointer rounded-lg px-5 py-2 shadow-md focus:outline-none mb-2`}
              >
                <div className="flex items-center justify-between w-full">
                  <div className="flex items-center">
                    <div className="text-sm">
                      <RadioGroup.Label as="p" className={`font-medium ${checked ? "text-white" : "text-gray-300"}`}>
                        仅挂单的
                      </RadioGroup.Label>
                    </div>
                  </div>
                  {checked && (
                    <div className="shrink-0 text-white">
                      <CheckIcon className="h-6 w-6" />
                    </div>
                  )}
                </div>
              </div>
            )}
          </RadioGroup.Option>
          <RadioGroup.Option value="all">
            {({ checked }) => (
              <div
                className={`${checked ? "bg-[#8e67e9] text-white" : "bg-gray-900 text-gray-300"} relative flex cursor-pointer rounded-lg px-5 py-2 shadow-md focus:outline-none`}
              >
                <div className="flex items-center justify-between w-full">
                  <div className="flex items-center">
                    <div className="text-sm">
                      <RadioGroup.Label as="p" className={`font-medium ${checked ? "text-white" : "text-gray-300"}`}>
                        显示全部
                      </RadioGroup.Label>
                    </div>
                  </div>
                  {checked && (
                    <div className="shrink-0 text-white">
                      <CheckIcon className="h-6 w-6" />
                    </div>
                  )}
                </div>
              </div>
            )}
          </RadioGroup.Option>
        </RadioGroup>
      </div>

      <div className="space-y-4">
        <h2 className="text-lg font-semibold flex items-center">
          <Eye className="h-5 w-5 mr-2" />
          系列
        </h2>
        <div className="relative">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" />
          <input
            type="text"
            placeholder="搜索你的系列"
            className="w-full pl-9 bg-gray-900 border border-gray-700 rounded-md py-2 px-3 text-sm focus:outline-none focus:border-[#8e67e9]"
          />
        </div>
        <div className="text-sm text-gray-400">
          <div className="flex justify-between py-2">
            <span>系列</span>
            <span>底价</span>
            <span>价值</span>
            <span>已挂单</span>
          </div>
          <div className="text-center py-4">未找到系列。</div>
        </div>
      </div>
    </div>
  )
}

function formatAddress(addr: string | undefined) {
  if (!addr) return "0x0000****0000";
  return `${addr.slice(0, 6)}***${addr.slice(-6)}`;
}

function CheckIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="none" {...props}>
      <circle cx={12} cy={12} r={12} fill="#fff" opacity="0.2" />
      <path d="M7 13l3 3 7-7" stroke="#fff" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

