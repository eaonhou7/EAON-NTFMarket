import { request } from './request';

function GetLoginMessage(address: string) {
    return request.get(`/user/${address}/login-message`);
};

function GetSigStatus(address: string) {
    return request.get(`/user/${address}/sig-status`);
};

type LoginParams = {
    chain_id: number;
    message: string;
    signature: string;
    address: string;
}

function Login(params: LoginParams) {
    return request.post(`/user/login`, params);
};

// eslint-disable-next-line import/no-anonymous-default-export
export default {
    GetLoginMessage,
    GetSigStatus,
    Login
}

