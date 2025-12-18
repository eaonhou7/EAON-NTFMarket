import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@headlessui/react"
// import { Label } from "@/components/ui/label"

const traits = [
  { name: "Background", count: 70 },
  { name: "Body", count: 32 },
  { name: "Costume", count: 54 },
  { name: "Eye", count: 44 },
  { name: "Front design", count: 37 },
  { name: "FRONT Logo", count: 33 },
  { name: "Hair", count: 225 },
  { name: "SpecialOption", count: 36 },
]

export function TraitFilters() {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <div className="flex items-center space-x-2">
          <Checkbox id="buy-now" />
          <label htmlFor="buy-now">仅显立即购买</label>
        </div>
        <div className="flex items-center space-x-2">
          <Checkbox id="show-all" />
          <label htmlFor="show-all">展示全部</label>
        </div>
      </div>

      <Accordion type="multiple" className="w-full">
        {traits.map((trait) => (
          <AccordionItem key={trait.name} value={trait.name.toLowerCase()}>
            <AccordionTrigger>
              {trait.name} ({trait.count})
            </AccordionTrigger>
            <AccordionContent>
              <div className="space-y-2 pt-2">
                {Array.from({ length: 3 }).map((_, i) => (
                  <div key={i} className="flex items-center space-x-2">
                    <Checkbox id={`${trait.name}-${i}`} />
                    <Label htmlFor={`${trait.name}-${i}`}>Trait Option {i + 1}</Label>
                  </div>
                ))}
              </div>
            </AccordionContent>
          </AccordionItem>
        ))}
      </Accordion>
    </div>
  )
}

