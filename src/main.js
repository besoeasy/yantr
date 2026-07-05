import './assets/main.css'

import { createApp } from 'vue'

import App from './App.vue'

import router from './router.js'

import i18n from './i18n/index.js'
import { installYantrFetchAuth } from './utils/fetchInterceptor.js'

installYantrFetchAuth()

const app = createApp(App)

app.use(router)

app.use(i18n)

app.mount('#app')
