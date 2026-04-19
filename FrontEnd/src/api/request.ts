import axios from 'axios'

const request = axios.create({
    // 你已配置 vite proxy，这里用相对路径即可
    baseURL: '/',
    timeout: 10000
})

request.interceptors.response.use(
    (res) => res.data,
    (err) => Promise.reject(err)
)

export default request