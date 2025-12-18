'use client'

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'

const chartData = [
    { time: '2024-12-21', value: 12, value2: 8 },
    { time: '2024-12-25', value: 11.8, value2: 7.8 },
    { time: '2024-12-29', value: 11.5, value2: 7.9 },
    { time: '2025-01-02', value: 12.2, value2: 7.5 },
    { time: '2025-01-06', value: 13.5, value2: 7.2 },
    { time: '2025-01-10', value: 15.8, value2: 6.8 },
    { time: '2025-01-14', value: 17.3, value2: 6.5 },
    { time: '2025-01-18', value: 18.9, value2: 6.2 },
]

export default function Trending() {
    return (
        <div className="flex flex-col gap-4 px-6 pt-6">
            <div className="flex items-center justify-between">
                <div className="flex gap-2">
                    {['1-3', '4-10', '11-20', '21-50', '51+'].map((range) => (
                        <label key={range} className="flex items-center gap-1.5">
                            <input
                                type="checkbox"
                                className="rounded border-gray-600 bg-gray-700 text-[#b599fd]"
                                defaultChecked={range === '4-10' || range === '11-20'}
                            />
                            <span className="text-sm text-gray-300">{range}</span>
                        </label>
                    ))}
                </div>
                <div className="flex gap-2">
                    {['1个月', '6个月', '1年', '2年', '5年', '所有', '每日'].map((period) => (
                        <button
                            key={period}
                            className={`px-3 py-1 text-sm rounded ${
                                period === '1个月'
                                    ? 'bg-[#b599fd] text-white'
                                    : 'text-gray-300 hover:bg-gray-700'
                            }`}
                        >
                            {period}
                        </button>
                    ))}
                </div>
            </div>
            <div className="w-full h-[400px]">
                <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={chartData}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#2e2e2e" />
                        <XAxis 
                            dataKey="time" 
                            stroke="#999" 
                            tick={{ fill: '#999' }}
                        />
                        <YAxis 
                            stroke="#999"
                            tick={{ fill: '#999' }}
                        />
                        <Tooltip 
                            contentStyle={{ 
                                backgroundColor: '#1e222d',
                                border: '1px solid #2e2e2e',
                                color: '#999'
                            }}
                        />
                        <Line 
                            type="monotone" 
                            dataKey="value" 
                            stroke="#b599fd" 
                            strokeWidth={2}
                            dot={false}
                        />
                        <Line 
                            type="monotone" 
                            dataKey="value2" 
                            stroke="#4c9fff" 
                            strokeWidth={2}
                            dot={false}
                        />
                    </LineChart>
                </ResponsiveContainer>
            </div>
        </div>
    )
}