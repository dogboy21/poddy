<script>
import axios from 'axios'

export default {
    data() {
        return {
            tabValue: 1,
            sessions: [],
            providers: [],
            workspaces: {},
            repositoryInfo: null,
        }
    },
    methods: {
        startLogin(provider) {
            location.replace('/oauth/auth/' + provider)
        },
        logout(provider) {
            location.replace('/oauth/logout/' + provider)
        },
        getRepositoryInfo() {
            let hash = location.hash
            if (!hash) {
                return null
            }

            hash = hash.substring(1)
            if (!hash.startsWith('http')) {
                return null
            }

            let url = new URL(hash)
            let path = url.pathname.substring(1)
            
            if (!path.includes('/-/tree/')) {
                return null
            }

            let repoParts = path.split('/-/tree/')
            if (repoParts.length !== 2) {
                return null
            }

            let project = repoParts[0]
            let branch = repoParts[1].slice(0, -1)

            return {
                url: hash,
                host: url.host,
                project: project,
                branch: branch,
            }
        },
        startWorkspaceCreation() {
            axios.post('/api/v1/workspaces', this.repositoryInfo)
                .then(creationResp => {
                    location.replace(location.href.substr(0, window.location.href.indexOf('#')))
                })
                .catch(err => {
                    console.log(err)
                    $vaToast.error({ message: 'Failed to create workspace', closeable: false })
                })
        },
        deleteWorkspace(workspace) {
            axios.delete('/api/v1/workspaces/' + workspace.provider + '/' + workspace.name)
                .then(resp => {
                    $vaToast.init({ message: 'Workspace ' + workspace.name + ' deleted', closeable: false })
                })
                .catch(err => {
                    console.error(err)
                    $vaToast.init({ message: 'Failed to delete workspace', closeable: false })
                })
        }
    },
    mounted() {
        axios.get('/oauth/providers')
            .then(providersResp => {
                this.providers = providersResp.data
                return axios.get('/api/v1/self')
            })
            .then(selfResp => {
                this.sessions = selfResp.data
                if (selfResp.data.length === 0) return null
                return axios.get('/api/v1/workspaces')
            })
            .then(workspacesResp => {
                if (!workspacesResp) return;

                this.workspaces = workspacesResp.data
                this.repositoryInfo = this.getRepositoryInfo()
                if (this.repositoryInfo) {
                    this.startWorkspaceCreation()
                }
            })
            .catch(err => {
                console.error(err)
                $vaToast.init({ message: 'Failed to load data', closeable: false })
            })
    },
    computed: {
        filteredProviders() {
            return this.providers.filter(provider => {
                return this.sessions.filter(session => {
                    return session.provider === provider.id
                }).length === 0
            })
        },
        hasWorkspaces() {
            return Object.keys(this.workspaces).length !== 0
        },
        workspacesList() {
            return Object.entries(this.workspaces)
                .flatMap(entry => entry[1].map(workspace => {
                    return {
                        ...workspace,
                        provider: entry[0],
                    }
                }))
        }
    }
}
</script>


<template>
    <div id="app-container" class="flex justify--center align--center">
        <div class="flex md4">
            <va-card>
                <va-card-title>Poddy</va-card-title>

                <va-card-content v-if="repositoryInfo === null">
                    <va-tabs v-model="tabValue">
                        <template #tabs>
                            <va-tab>Sessions</va-tab>
                            <va-tab v-if="sessions.length > 0">Workspaces</va-tab>
                        </template>

                        <template #default v-if="tabValue === 1">
                            <va-list v-if="sessions.length > 0">
                                <va-list-label>Current Sessions</va-list-label>

                                <va-list-item v-for="session in sessions" v-bind:key="session.host">
                                    <va-list-item-section avatar>
                                        <va-avatar v-bind:src="session.user.avatar_url" v-bind:title="session.user.display_name" />
                                    </va-list-item-section>

                                    <va-list-item-section>
                                        <va-list-item-label>{{ session.user.display_name }}</va-list-item-label>
                                        <va-list-item-label caption>{{ session.host }}</va-list-item-label>
                                    </va-list-item-section>

                                    <va-list-item-section icon>
                                        <va-button @click="logout(session.provider)">Logout</va-button>
                                    </va-list-item-section>
                                </va-list-item>
                            </va-list>

                            <va-list v-if="filteredProviders.length > 0">
                                <va-list-label>Available Providers</va-list-label>

                                <va-list-item v-for="provider in filteredProviders" v-bind:key="provider.id">
                                    <va-list-item-section avatar>
                                        <va-avatar>{{ provider.host.substring(0, 1).toUpperCase() }}</va-avatar>
                                    </va-list-item-section>

                                    <va-list-item-section>
                                        <va-list-item-label>{{ provider.host }}</va-list-item-label>
                                    </va-list-item-section>

                                    <va-list-item-section icon>
                                        <va-button @click="startLogin(provider.id)">Login</va-button>
                                    </va-list-item-section>
                                </va-list-item>
                            </va-list>
                        </template>

                        <template #default v-else-if="tabValue === 2">
                            <va-list v-if="hasWorkspaces">
                                <va-list-label>Active Workspaces</va-list-label>

                                <va-list-item v-for="workspace in workspacesList" v-bind:key="workspace.name">
                                    <va-list-item-section avatar>
                                        <va-avatar>{{ workspace.name.substring(0, 1).toUpperCase() }}</va-avatar>
                                    </va-list-item-section>

                                    <va-list-item-section>
                                        <va-list-item-label>{{ workspace.name }}</va-list-item-label>
                                        <va-list-item-label caption><a :href="'https://' + workspace.url" target="_blank" style="color:inherit;">{{ workspace.url }}</a></va-list-item-label>
                                    </va-list-item-section>

                                    <va-list-item-section icon>
                                        <va-button @click="deleteWorkspace(workspace)">Delete</va-button>
                                    </va-list-item-section>
                                </va-list-item>
                            </va-list>

                            <p v-else>You don't have any active workspaces currently</p>
                        </template>
                    </va-tabs>
                </va-card-content>

                <va-card-content v-else>
                    <p>Please wait while your workspace for {{ repositoryInfo.project }} is being created...</p>
                </va-card-content>
            </va-card>
        </div>


    </div>
</template>

<style>
body {
    background: #333;
    font-family: 'Open Sans', sans-serif;
}

html, body, #app, #app-container {
    width: 100%;
    height: 100%;
}

#app-container {
    display: flex;
}

.va-tabs__content {
    box-sizing: border-box;
    padding: 12px;
}

</style>
