import { createAuthClient } from '@remark42/api/clients/auth'
import { createPublicClient } from '@remark42/api/clients/public'

const siteId = 'remark'
const baseUrl = ''

export let publicApi = createPublicClient({ siteId, baseUrl })
export let authApi = createAuthClient({ siteId, baseUrl })
