'use client'


import { CollectionsProvider } from "./hooks/context"
import { SubNav } from "./components/sub-nav"
import Main from "./components/main"


export default function Collections() {
   

    return (
        <CollectionsProvider>
            <div>
                <SubNav />
                <Main />
            </div>
        </CollectionsProvider>
    )
}