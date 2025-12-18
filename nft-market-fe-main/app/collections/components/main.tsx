
import { useContext } from "react";
import { CollectionsContext } from "../hooks/context";
import Collections from "./collections";
import Trending from "./trending";
export default function Main() {

    const { activeTab } = useContext(CollectionsContext)

    // if (activeTab === 'collections') {
    //     return <Collections />
    if (activeTab === 'trending') {
        return <Collections type={'trending'}/>
    }
    return (
        <Collections type={''}/>
    )
}