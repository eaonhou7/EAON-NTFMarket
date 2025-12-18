import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

// 定义状态类型
interface GlobalState {
  theme: 'light' | 'dark'
  language: string
  walletAddress: string

  chain_id: string | number

  collection_address: string
  collection: Record<string, any> | null
  // 其他全局状态...
}

// 定义初始状态
const initialState: GlobalState = {
  theme: 'light',
  language: 'zh',
  walletAddress: '',
  chain_id: 11155111,
  collection_address: '0x0000000000000000000000000000000000000000',
  collection: null,
}

// 创建查询键
const GLOBAL_STATE_KEY = 'globalState'

export function useGlobalState() {
  const queryClient = useQueryClient()

  // 获取状态
  const { data: state = initialState } = useQuery({
    queryKey: [GLOBAL_STATE_KEY],
    queryFn: () => {
      // 这里可以从 localStorage 或 API 获取初始状态
      const savedState = localStorage.getItem(GLOBAL_STATE_KEY)
      return savedState ? JSON.parse(savedState) : initialState
    },
    staleTime: Infinity, // 永不过期
  })

  // 更新状态
  const { mutate: setState } = useMutation({
    mutationFn: (newState: Partial<GlobalState>) => {
      const updatedState = { ...state, ...newState }
      localStorage.setItem(GLOBAL_STATE_KEY, JSON.stringify(updatedState))
      return updatedState
    },
    onSuccess: (newState) => {
      // 更新缓存
      queryClient.setQueryData([GLOBAL_STATE_KEY], newState)
    },
  })

  return {
    state,
    setState,
  }
}