import { request } from "./request";

interface ActivityFilter {
    filter_ids: number[];            // 链 ID 数组
    collection_addresses: string[];   // NFT 合约地址数组
    token_id?: string;                // NFT token ID
    user_addresses?: string[];        // 用户钱包地址数组
    event_types?: string[];           // 事件类型数组
    page: number;                    // 页码
    page_size: number;               // 每页条数
}

function GetActivity(params: ActivityFilter) {
    // return request.get('/activity')
    return request.get('/activities', {
        params: {
            filters: JSON.stringify(params),
        },
    });
}

export default {
    GetActivity
}