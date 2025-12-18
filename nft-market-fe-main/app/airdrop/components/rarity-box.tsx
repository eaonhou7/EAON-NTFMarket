"use client"

import { useEffect, useRef } from "react"

interface RarityBoxProps {
  type: "uncommon" | "rare" | "legendary" | "mythical"
  color: string
  borderColor: string
  count: string
}

export function RarityBox({ type, color, borderColor, count }: RarityBoxProps) {
  const boxRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const box = boxRef.current
    if (!box) return

    let frame: number
    let rotation = 0

    const animate = () => {
      rotation += 0.5
      box.style.transform = `rotateY(${rotation}deg)`
      frame = requestAnimationFrame(animate)
    }

    frame = requestAnimationFrame(animate)

    return () => {
      cancelAnimationFrame(frame)
    }
  }, [])

  return (
    <div className="flex flex-col items-center space-y-4">
      <div
        ref={boxRef}
        className={`
          w-40 h-40 relative
          transform-gpu preserve-3d
          cursor-pointer
          transition-transform duration-500
        `}
      >
        {/* Front */}
        <div className={`absolute inset-0 bg-gradient-to-br ${color} border ${borderColor} transform-gpu`} />
        {/* Back */}
        <div
          className={`absolute inset-0 bg-gradient-to-br ${color} border ${borderColor} transform-gpu -translate-z-40`}
        />
        {/* Left */}
        <div
          className={`absolute inset-0 bg-gradient-to-br ${color} border ${borderColor} transform-gpu -translate-x-20 rotate-y-90`}
        />
        {/* Right */}
        <div
          className={`absolute inset-0 bg-gradient-to-br ${color} border ${borderColor} transform-gpu translate-x-20 -rotate-y-90`}
        />
        {/* Top */}
        <div
          className={`absolute inset-0 bg-gradient-to-br ${color} border ${borderColor} transform-gpu -translate-y-20 rotate-x-90`}
        />
        {/* Bottom */}
        <div
          className={`absolute inset-0 bg-gradient-to-br ${color} border ${borderColor} transform-gpu translate-y-20 -rotate-x-90`}
        />
      </div>
      <div className="text-center">
        <div className="uppercase font-mono tracking-wider">{type}</div>
        <div
          className={`
          text-2xl font-bold
          ${type === "uncommon" && "text-orange-500"}
          ${type === "rare" && "text-red-500"}
          ${type === "legendary" && "text-purple-500"}
          ${type === "mythical" && "text-green-500"}
        `}
        >
          {count}
        </div>
      </div>
    </div>
  )
}

