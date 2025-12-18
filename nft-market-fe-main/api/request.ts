import axios from 'axios';

const isProd = process.env.NODE_ENV === 'production';

export const request = axios.create({
    // baseURL: isProd ? '/api/v1' : 'http://nft.rcc-tec.xyz',
    baseURL: '/api/v1',
    timeout: 50000,
    headers: {
        'Content-Type': 'application/json',
    },
});

request.interceptors.request.use(config => {
    return config;
});

request.interceptors.response.use(response => {
    return response.data.data;
});
