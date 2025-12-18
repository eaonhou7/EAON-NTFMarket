"use client"

import { Tab } from "@headlessui/react"

export function SeasonTabs() {
  return (
    <Tab.Group defaultIndex={0}>
      <Tab.List className="flex space-x-4">
        <Tab
          className={({ selected }) => `
          px-6 py-2 rounded text-sm font-mono uppercase
          ${selected ? "text-[#CEC5FD] border-b-2 border-[#CEC5FD]" : "text-gray-400"}
        `}
        >
          <div className="flex items-center space-x-2">
            <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
              <path
                fillRule="evenodd"
                d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                clipRule="evenodd"
              />
            </svg>
            <span>Season 1</span>
          </div>
        </Tab>
        <Tab
          className={({ selected }) => `
          px-6 py-2 rounded text-sm font-mono uppercase
          ${selected ? "text-[#CEC5FD] border-b-2 border-[#CEC5FD]" : "text-gray-400"}
        `}
        >
          <span>敬请期待～</span>
        </Tab>
      </Tab.List>
    </Tab.Group>
  )
}

