"use client"

import React from "react"
import { Switch as HeadlessSwitch } from "@headlessui/react"
import { cn } from "@/lib/utils"

const Switch = React.forwardRef<
  React.ElementRef<typeof HeadlessSwitch>,
  React.ComponentPropsWithoutRef<typeof HeadlessSwitch>
>(({ className, ...props }, ref) => (
  <HeadlessSwitch
    ref={ref}
    className={cn(
      "relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500",
      props.checked ? "bg-indigo-600" : "bg-gray-200",
      className,
    )}
    {...props}
  >
    <span className="sr-only">Enable</span>
    <span
      className={cn(
        "inline-block h-4 w-4 transform rounded-full bg-white transition-transform",
        props.checked ? "translate-x-6" : "translate-x-1",
      )}
    />
  </HeadlessSwitch>
))
Switch.displayName = "Switch"

export { Switch }

