"use client"

import React from "react"
import { Tab } from "@headlessui/react"
import { cn } from "@/lib/utils"

const Tabs = Tab.Group

const TabsList = React.forwardRef<React.ElementRef<typeof Tab.List>, React.ComponentPropsWithoutRef<typeof Tab.List>>(
  ({ className, ...props }, ref) => (
    <Tab.List
      ref={ref}
      className={cn(
        "inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground",
        className,
      )}
      {...props}
    />
  ),
)
TabsList.displayName = "TabsList"

const TabsTrigger = React.forwardRef<React.ElementRef<typeof Tab>, React.ComponentPropsWithoutRef<typeof Tab>>(
  ({ className, ...props }, ref) => (
    <Tab
      ref={ref}
      className={({ selected }) =>
        cn(
          "inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
          selected ? "bg-background text-foreground shadow" : "hover:bg-muted hover:text-foreground",
          className,
        )
      }
      {...props}
    />
  ),
)
TabsTrigger.displayName = "TabsTrigger"

const TabsContent = React.forwardRef<
  React.ElementRef<typeof Tab.Panel>,
  React.ComponentPropsWithoutRef<typeof Tab.Panel>
>(({ className, ...props }, ref) => (
  <Tab.Panel
    ref={ref}
    className={cn(
      "mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
      className,
    )}
    {...props}
  />
))
TabsContent.displayName = "TabsContent"

export { Tabs, TabsList, TabsTrigger, TabsContent }

