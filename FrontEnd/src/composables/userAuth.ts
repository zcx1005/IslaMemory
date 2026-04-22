import { computed, ref } from 'vue'
import {
    getMyProfile,
    loginAccount,
    registerAccount,
    type AuthUser,
    type LoginPayload,
    type RegisterPayload,
} from '@/api/auth'

const TOKEN_KEY = 'isla_token'
const USER_KEY = 'isla_user'

const token = ref(localStorage.getItem(TOKEN_KEY) || '')
const user = ref<AuthUser | null>(readUser())
const authLoading = ref(false)

const showAuthModal = ref(false)
const activeAuthTab = ref<'login' | 'register'>('login')

function readUser(): AuthUser | null {
    const raw = localStorage.getItem(USER_KEY)
    if (!raw) return null
    try {
        return JSON.parse(raw)
    } catch {
        localStorage.removeItem(USER_KEY)
        return null
    }
}

function persistAuth(nextToken: string, nextUser: AuthUser | null) {
    token.value = nextToken
    user.value = nextUser

    if (nextToken) {
        localStorage.setItem(TOKEN_KEY, nextToken)
    } else {
        localStorage.removeItem(TOKEN_KEY)
    }

    if (nextUser) {
        localStorage.setItem(USER_KEY, JSON.stringify(nextUser))
    } else {
        localStorage.removeItem(USER_KEY)
    }
}

async function register(payload: RegisterPayload) {
    authLoading.value = true
    try {
        return await registerAccount(payload)
    } finally {
        authLoading.value = false
    }
}

async function login(payload: LoginPayload) {
    authLoading.value = true
    try {
        const res = await loginAccount(payload)
        if (res.code === 200 && res.data?.token) {
            persistAuth(res.data.token, res.data.user)
        }
        return res
    } finally {
        authLoading.value = false
    }
}

async function restoreSession() {
    if (!token.value) return
    try {
        const res = await getMyProfile()
        if (res.code === 200 && res.data) {
            persistAuth(token.value, res.data)
            return
        }
    } catch {
        // token 失效时静默清理
    }
    persistAuth('', null)
}

function logout() {
    persistAuth('', null)
}


function updateUserProfile(partial: Partial<AuthUser>) {
    if (!user.value) return
    const merged = { ...user.value, ...partial }
    persistAuth(token.value, merged)
}

function openAuthModal(tab: 'login' | 'register' = 'login') {
    activeAuthTab.value = tab
    showAuthModal.value = true
}

function closeAuthModal() {
    showAuthModal.value = false
}

export function getAuthToken() {
    return token.value
}

export function useAuth() {
    return {
        token,
        user,
        authLoading,
        showAuthModal,
        activeAuthTab,
        isLoggedIn: computed(() => !!token.value),
        register,
        login,
        logout,
        restoreSession,
        updateUserProfile,
        openAuthModal,
        closeAuthModal,
    }
}