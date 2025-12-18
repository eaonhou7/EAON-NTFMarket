import { useRouter } from "next/navigation";
import { createContext, useEffect, useState } from "react"

// 定义上下文类型
type CollectionsContextType = {
    activeTab: string;
    setActiveTab: (tab: string) => void;
  };

export const CollectionsContext = createContext<CollectionsContextType>({
    activeTab: 'collections',
    setActiveTab: (tab: string) => {
       console.log(tab, '=tab')
    },
})

export const CollectionsProvider = ({ children }: { children: React.ReactNode }) => {
    const router = useRouter()

    const [activeTab, setActiveTab] = useState('');


    useEffect(() => {
        const initialTab = window.location.hash.slice(1)
        if(!initialTab) {
            
            router.push('#trending')
            setActiveTab('trending')
        } else {
            setActiveTab(initialTab)
        }
    }, [])

    return (
        <CollectionsContext.Provider value={{ activeTab, setActiveTab }}>
            {children}
        </CollectionsContext.Provider>
    );
}