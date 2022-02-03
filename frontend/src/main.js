import { createApp } from 'vue'
import App from './App.vue'

import { VuesticPlugin } from 'vuestic-ui';

const app = createApp(App)
app.use(VuesticPlugin)

app.mount('#app')
