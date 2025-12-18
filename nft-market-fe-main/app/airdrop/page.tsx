import { SeasonTabs } from "./components/season-tabs"
import { RarityBoxes } from "./components/rarity-boxes"
import { RccLogo } from "./components/rcc-logo"

export default function Page() {
  return (
    <div className="min-h-screen bg-black text-white p-8">
      <div className="max-w-6xl mx-auto relative">
        <SeasonTabs />

        <div className="mt-16 text-center space-y-12">
          <h1 className="text-6xl font-bold tracking-wider text-primary animate-glow">RCC SEASON ONE FINALE</h1>

          <p className="text-xl font-mono">
          RCC Season 1 is officially over. It's time to take Blur to the next level - Season 2 now begins.
          </p>

          <RarityBoxes />

        </div>

      </div>
    </div>
  )
}

