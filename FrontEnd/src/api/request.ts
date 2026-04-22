import axios from 'axios'
import { getAuthToken } from '@/composables/userAuth'

const request = axios.create({
    // 你已配置 vite proxy，这里用相对路径即可
    baseURL: '/',
    timeout: 10000
})

request.interceptors.request.use((config) => {
    const token = getAuthToken()
    if (token) {
        config.headers = config.headers || {}
        config.headers.Authorization = `Bearer ${token}`
    }
    return config
})

request.interceptors.response.use(
    (res) => res.data,
    (err) => {
        const msg = err?.response?.data?.msg || err?.message || '请求失败'
        return Promise.reject(new Error(msg))
    }
)

export default request