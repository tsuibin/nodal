/**
 * 模块 10：Demo 画布种子。
 *
 * POST /api/demo/seed 幂等：已有画布返回 created=0。
 */
import { request } from './http'
import type { DemoSeedResult } from '../types/api'

export const demoApi = {
  seed(): Promise<DemoSeedResult> {
    return request<DemoSeedResult>({ url: '/demo/seed', method: 'POST' })
  },
}
