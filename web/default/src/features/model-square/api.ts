/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { api } from '@/lib/api'
import type {
  ModelSquareData,
  ModelSquareParams,
  ModelSquareResponse,
} from './types'

export async function getModelSquare(
  params: ModelSquareParams = {}
): Promise<ModelSquareData> {
  const paths = ['/api/v2/model-square', '/api/v2/model_square']
  let lastError: Error | null = null

  for (const path of paths) {
    try {
      const response = await api.get<ModelSquareResponse>(path, {
        params,
        disableDuplicate: true,
        skipBusinessError: true,
        skipErrorHandler: true,
      } as Parameters<typeof api.get>[1] & {
        disableDuplicate: boolean
        skipBusinessError: boolean
        skipErrorHandler: boolean
      })
      const result = response.data
      if (!result.success) {
        lastError = new Error(
          result.message || 'Unable to load model square data'
        )
        continue
      }
      return result.data
    } catch (error) {
      lastError =
        error instanceof Error
          ? error
          : new Error('Unable to load model square data')
    }
  }

  if (import.meta.env.DEV) {
    try {
      return await getModelSquareViaDevFallback(params)
    } catch (error) {
      lastError =
        error instanceof Error
          ? error
          : new Error('Unable to load model square data')
    }
  }

  throw lastError || new Error('Unable to load model square data')
}

async function getModelSquareViaDevFallback(
  params: ModelSquareParams
): Promise<ModelSquareData> {
  const searchParams = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value) searchParams.set(key, value)
  }
  const query = searchParams.toString()
  const serverUrl =
    import.meta.env.VITE_REACT_APP_SERVER_URL || 'http://localhost:3010'
  const response = await fetch(
    `${serverUrl.replace(/\/$/, '')}/api/v2/model_square${query ? `?${query}` : ''}`,
    {
      headers: {
        'Cache-Control': 'no-store',
      },
    }
  )
  if (!response.ok) {
    throw new Error(`Unable to load model square data: ${response.status}`)
  }
  const result = (await response.json()) as ModelSquareResponse
  if (!result.success) {
    throw new Error(result.message || 'Unable to load model square data')
  }
  return result.data
}
