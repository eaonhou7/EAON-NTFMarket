import { AlertTriangle } from "lucide-react"

export function WarningBanner() {
  return (
    <div className="bg-[#CEC5FD33] text-[#CEC5FD] px-4 py-2 flex items-center justify-center">
      <AlertTriangle className="h-4 w-4 mr-2" />
      <span>You are not connected to Ethereum. Switch Network in your wallet to continue.</span>
    </div>
  )
}

