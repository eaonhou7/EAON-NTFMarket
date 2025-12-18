"use client"

import React from "react"
import { Checkbox as HeadlessCheckbox } from "@headlessui/react"
import { Check } from "lucide-react"
import { cn } from "@/lib/utils"

const Checkbox = React.forwardRef<
  React.ElementRef<typeof HeadlessCheckbox>,
  React.ComponentPropsWithoutRef<typeof HeadlessCheckbox>
>(({ className, ...props }, ref) => (
  <HeadlessCheckbox
    ref={ref}
    className={cn(
      "peer h-4 w-4 shrink-0 rounded-sm border border-primary ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground",
      className,
    )}
    {...props}
  >
    {({ checked }) => (
      <div className="flex items-center justify-center">{checked && <Check className="h-4 w-4" />}</div>
    )}
  </HeadlessCheckbox>
))
Checkbox.displayName = "Checkbox"

export { Checkbox }

