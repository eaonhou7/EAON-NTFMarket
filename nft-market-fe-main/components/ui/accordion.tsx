// @ts-nocheck

"use client"

import React from 'react'
import { Disclosure } from '@headlessui/react'
import { ChevronDown } from 'lucide-react'
import { cn } from "@/lib/utils"

const Accordion = ({ children, ...props }: React.ComponentProps<"div">) => (
  <div {...props}>{children}</div>
)

// const AccordionItem = Disclosure
const AccordionItem = React.forwardRef<
  React.ElementRef<"div">,
  React.ComponentPropsWithoutRef<typeof Disclosure>
>((props, ref) => (
  <Disclosure as="div" ref={ref} {...props} />
))
AccordionItem.displayName = "AccordionItem"


const AccordionTrigger = React.forwardRef<
  React.ElementRef<typeof Disclosure.Button>,
  React.ComponentPropsWithoutRef<typeof Disclosure.Button>
>(({ className, children, ...props }, ref) => (
  <Disclosure.Button
    as="button"
    ref={ref}
    className={cn(
      "flex w-full items-center justify-between py-4 font-medium transition-all hover:underline [&[data-state=open]>svg]:rotate-180",
      className
    )}
    {...props}
  >

    {/* @ts-ignore */}
    {children}
    <ChevronDown className="h-4 w-4 shrink-0 transition-transform duration-200" />
  </Disclosure.Button>
))
AccordionTrigger.displayName = "AccordionTrigger"

const AccordionContent = React.forwardRef<
  React.ElementRef<typeof Disclosure.Panel>,
  React.ComponentPropsWithoutRef<typeof Disclosure.Panel>
>(({ className, children, ...props }, ref) => (
  <Disclosure.Panel
    as="div"
    ref={ref}
    className={cn(
      "overflow-hidden text-sm transition-all data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down",
      className
    )}
    {...props}
  >
    {/* @ts-ignore */}
    {children}
  </Disclosure.Panel>
))
AccordionContent.displayName = "AccordionContent"

export { Accordion, AccordionItem, AccordionTrigger, AccordionContent }
