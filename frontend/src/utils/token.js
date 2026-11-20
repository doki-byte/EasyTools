import { getItem, setItem, removeItem } from './storageManager'

const TokenKey = 'EasyTools-Token'

export function getToken() {
    return getItem(TokenKey, null)
}

export function setToken(token) {
    return setItem(TokenKey, token)
}

export function removeToken() {
    removeItem(TokenKey)
}

