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
import type { Modality, ModelCapability, PricingModel } from '../types'
import { hashStringToSeed, seededRandom } from './seed'

// ----------------------------------------------------------------------------
// Model metadata inference
// ----------------------------------------------------------------------------
//
// The backend does not currently return `context_length`, `max_output_tokens`,
// `knowledge_cutoff`, `release_date`, `parameter_count`, or modality/capability
// flags for a model. Until it does, we infer values client-side using:
//   1. KNOWN_MODEL_CONTEXT — exact model name match for ~100 well-known models
//   2. Regex patterns — for model families and name-hint based inference
//   3. Deterministic random bucket — only for truly unknown models
//
// Values seeded from the model name so every render of the same model is stable.
//
// When the backend starts returning these fields, callers should prefer the
// explicit values on `model.*` and only fall back to the inferred ones.

const TEXT_INPUT_ENDPOINTS = new Set([
  'openai',
  'openai-response',
  'anthropic',
  'gemini',
  'embeddings',
  'jina-rerank',
])

const IMAGE_OUTPUT_ENDPOINTS = new Set(['image-generation'])
const VIDEO_OUTPUT_ENDPOINTS = new Set(['openai-video'])
const EMBEDDING_ENDPOINTS = new Set(['embeddings', 'jina-rerank'])

const REASONING_NAME_PATTERNS = [
  /^o[1-4](?:[-:_].+)?$/i,
  /reasoning/i,
  /thinking/i,
  /qwq/i,
  /deepseek-r\d/i,
  /grok.*-(?:thinking|reasoning)/i,
]

const VISION_NAME_PATTERNS = [
  /vision/i,
  /vl(?:[-_]|$)/i,
  /multimodal/i,
  /-omni/i,
]

const AUDIO_NAME_PATTERNS = [
  /audio/i,
  /whisper/i,
  /tts/i,
  /voice/i,
  /-realtime/i,
]

const VIDEO_NAME_PATTERNS = [/video/i, /sora/i, /veo/i, /kling/i, /pika/i]

const CODE_NAME_PATTERNS = [/code/i, /-coder/i]

const WEB_SEARCH_PATTERNS = [/web[-_ ]?search/i, /-online/i, /perplexity/i]

const KNOWLEDGE_CUTOFFS = [
  '2023-04',
  '2023-10',
  '2023-12',
  '2024-04',
  '2024-06',
  '2024-08',
  '2024-10',
  '2024-12',
  '2025-02',
  '2025-04',
  '2025-08',
]

const PARAM_BUCKETS = [
  '1.5B',
  '3B',
  '7B',
  '8B',
  '14B',
  '32B',
  '70B',
  '120B',
  '405B',
]

const CONTEXT_BUCKETS = [
  8_192, 16_384, 32_768, 65_536, 128_000, 200_000, 1_000_000,
]
const MAX_OUTPUT_BUCKETS = [2_048, 4_096, 8_192, 16_384, 32_768, 65_536]

// Exact model name -> { context, maxOutput } mapping for known models.
// Values are approximate; prefer backend-provided values when available.
const KNOWN_MODEL_CONTEXT: Record<string, { context: number; maxOutput: number }> = {
  // OpenAI
  'gpt-4': { context: 8_192, maxOutput: 8_192 },
  'gpt-4-0613': { context: 8_192, maxOutput: 8_192 },
  'gpt-4-1106-preview': { context: 128_000, maxOutput: 4_096 },
  'gpt-4-0125-preview': { context: 128_000, maxOutput: 4_096 },
  'gpt-4-32k': { context: 32_768, maxOutput: 32_768 },
  'gpt-4-32k-0613': { context: 32_768, maxOutput: 32_768 },
  'gpt-4-turbo-preview': { context: 128_000, maxOutput: 4_096 },
  'gpt-4-turbo': { context: 128_000, maxOutput: 4_096 },
  'gpt-4-turbo-2024-04-09': { context: 128_000, maxOutput: 4_096 },
  'gpt-4o': { context: 128_000, maxOutput: 16_384 },
  'gpt-4o-2024-05-13': { context: 128_000, maxOutput: 16_384 },
  'gpt-4o-2024-08-06': { context: 128_000, maxOutput: 16_384 },
  'gpt-4o-2024-11-20': { context: 128_000, maxOutput: 16_384 },
  'chatgpt-4o-latest': { context: 128_000, maxOutput: 16_384 },
  'gpt-4o-mini': { context: 128_000, maxOutput: 16_384 },
  'gpt-4o-mini-2024-07-18': { context: 128_000, maxOutput: 16_384 },
  'gpt-4.5-preview': { context: 128_000, maxOutput: 16_384 },
  'gpt-4.5-preview-2025-02-27': { context: 128_000, maxOutput: 16_384 },
  'gpt-4.1': { context: 128_000, maxOutput: 16_384 },
  'gpt-4.1-2025-04-14': { context: 128_000, maxOutput: 16_384 },
  'gpt-4.1-mini': { context: 128_000, maxOutput: 16_384 },
  'gpt-4.1-nano': { context: 128_000, maxOutput: 16_384 },
  'o1': { context: 128_000, maxOutput: 65_536 },
  'o1-2024-12-17': { context: 128_000, maxOutput: 65_536 },
  'o1-preview': { context: 128_000, maxOutput: 65_536 },
  'o1-preview-2024-09-12': { context: 128_000, maxOutput: 65_536 },
  'o1-mini': { context: 65_536, maxOutput: 65_536 },
  'o1-mini-2024-09-12': { context: 65_536, maxOutput: 65_536 },
  'o1-pro': { context: 128_000, maxOutput: 65_536 },
  'o1-pro-2025-03-19': { context: 128_000, maxOutput: 65_536 },
  'o3-mini': { context: 128_000, maxOutput: 65_536 },
  'o3-mini-2025-01-31': { context: 128_000, maxOutput: 65_536 },
  'o3-mini-high': { context: 128_000, maxOutput: 65_536 },
  'o3-mini-low': { context: 128_000, maxOutput: 65_536 },
  'o3-mini-medium': { context: 128_000, maxOutput: 65_536 },
  'o3': { context: 128_000, maxOutput: 65_536 },
  'o3-2025-04-16': { context: 128_000, maxOutput: 65_536 },
  'o3-pro': { context: 128_000, maxOutput: 65_536 },
  'o3-pro-2025-06-10': { context: 128_000, maxOutput: 65_536 },
  'o3-deep-research': { context: 128_000, maxOutput: 65_536 },
  'o3-deep-research-2025-06-26': { context: 128_000, maxOutput: 65_536 },
  'o4-mini': { context: 128_000, maxOutput: 65_536 },
  'o4-mini-2025-04-16': { context: 128_000, maxOutput: 65_536 },
  'o4-mini-deep-research': { context: 128_000, maxOutput: 65_536 },
  'o4-mini-deep-research-2025-06-26': { context: 128_000, maxOutput: 65_536 },
  'gpt-5': { context: 128_000, maxOutput: 65_536 },
  'gpt-5-2025-08-07': { context: 128_000, maxOutput: 65_536 },
  'gpt-5-chat-latest': { context: 128_000, maxOutput: 65_536 },
  'gpt-5-mini': { context: 128_000, maxOutput: 65_536 },
  'gpt-5-mini-2025-08-07': { context: 128_000, maxOutput: 65_536 },
  'gpt-5-nano': { context: 128_000, maxOutput: 65_536 },
  'gpt-5-nano-2025-08-07': { context: 128_000, maxOutput: 65_536 },
  'gpt-5-pro': { context: 128_000, maxOutput: 65_536 },
  'gpt-5-pro-2025-10-06': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.1': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.1-2025-11-13': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.1-chat-latest': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.2': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.2-2025-12-11': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.2-chat-latest': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.3-chat-latest': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.4': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.4-2026-03-05': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.4-pro': { context: 128_000, maxOutput: 65_536 },
  'gpt-5.4-pro-2026-03-05': { context: 128_000, maxOutput: 65_536 },
  'text-embedding-ada-002': { context: 8_192, maxOutput: 0 },
  'text-embedding-3-small': { context: 8_192, maxOutput: 0 },
  'text-embedding-3-large': { context: 8_192, maxOutput: 0 },
  // Claude
  'claude-3-sonnet-20240229': { context: 200_000, maxOutput: 4_096 },
  'claude-3-opus-20240229': { context: 200_000, maxOutput: 4_096 },
  'claude-3-haiku-20240307': { context: 200_000, maxOutput: 4_096 },
  'claude-3-5-haiku-20241022': { context: 200_000, maxOutput: 8_192 },
  'claude-haiku-4-5-20251001': { context: 200_000, maxOutput: 8_192 },
  'claude-3-5-sonnet-20240620': { context: 200_000, maxOutput: 8_192 },
  'claude-3-5-sonnet-20241022': { context: 200_000, maxOutput: 8_192 },
  'claude-3-7-sonnet-20250219': { context: 200_000, maxOutput: 8_192 },
  'claude-3-7-sonnet-20250219-thinking': { context: 200_000, maxOutput: 8_192 },
  'claude-sonnet-4-20250514': { context: 200_000, maxOutput: 16_384 },
  'claude-sonnet-4-20250514-thinking': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-20250514': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-20250514-thinking': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-1-20250805': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-1-20250805-thinking': { context: 200_000, maxOutput: 16_384 },
  'claude-sonnet-4-5-20250929': { context: 200_000, maxOutput: 16_384 },
  'claude-sonnet-4-5-20250929-thinking': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-5-20251101': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-5-20251101-thinking': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-6': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-6-max': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-6-high': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-6-medium': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-6-low': { context: 200_000, maxOutput: 16_384 },
  'claude-sonnet-4-6': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-7': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-7-max': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-7-xhigh': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-7-high': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-7-medium': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-7-low': { context: 200_000, maxOutput: 16_384 },
  'claude-opus-4-7-thinking': { context: 200_000, maxOutput: 16_384 },
  // DeepSeek
  'deepseek-chat': { context: 64_000, maxOutput: 8_192 },
  'deepseek-reasoner': { context: 64_000, maxOutput: 8_192 },
  'deepseek-v4-flash': { context: 64_000, maxOutput: 8_192 },
  'deepseek-v4-flash-none': { context: 64_000, maxOutput: 8_192 },
  'deepseek-v4-flash-max': { context: 64_000, maxOutput: 8_192 },
  'deepseek-v4-pro': { context: 128_000, maxOutput: 8_192 },
  'deepseek-v4-pro-none': { context: 128_000, maxOutput: 8_192 },
  'deepseek-v4-pro-max': { context: 128_000, maxOutput: 8_192 },
  // Qwen
  'qwen-turbo': { context: 128_000, maxOutput: 8_192 },
  'qwen-plus': { context: 128_000, maxOutput: 8_192 },
  'qwen-max': { context: 32_768, maxOutput: 8_192 },
  'qwen-max-longcontext': { context: 1_000_000, maxOutput: 8_192 },
  'qwq-32b': { context: 32_768, maxOutput: 8_192 },
  'qwen3-235b-a22b': { context: 32_768, maxOutput: 8_192 },
  // Mistral
  'open-mistral-7b': { context: 8_192, maxOutput: 8_192 },
  'open-mixtral-8x7b': { context: 8_192, maxOutput: 8_192 },
  'mistral-small-latest': { context: 32_768, maxOutput: 8_192 },
  'mistral-medium-latest': { context: 32_768, maxOutput: 8_192 },
  'mistral-large-latest': { context: 128_000, maxOutput: 32_768 },
  'mistral-embed': { context: 8_192, maxOutput: 0 },
  // Kimi / Moonshot
  'kimi-k2.5': { context: 128_000, maxOutput: 32_768 },
  'kimi-k2-0905-preview': { context: 128_000, maxOutput: 32_768 },
  'kimi-k2-turbo-preview': { context: 128_000, maxOutput: 32_768 },
  'kimi-k2-thinking': { context: 128_000, maxOutput: 32_768 },
  'kimi-k2-thinking-turbo': { context: 128_000, maxOutput: 32_768 },
  // MiniMax
  'MiniMax-M2.7': { context: 32_768, maxOutput: 8_192 },
  'MiniMax-M2.7-highspeed': { context: 32_768, maxOutput: 8_192 },
  'MiniMax-M2.1': { context: 32_768, maxOutput: 8_192 },
  'MiniMax-M2.1-highspeed': { context: 32_768, maxOutput: 8_192 },
  'MiniMax-M2': { context: 32_768, maxOutput: 8_192 },
  'MiniMax-M2.5': { context: 32_768, maxOutput: 8_192 },
  'MiniMax-M2.5-highspeed': { context: 32_768, maxOutput: 8_192 },
  // Cohere
  'command-a-03-2025': { context: 128_000, maxOutput: 4_096 },
  'command-r': { context: 4_096, maxOutput: 4_096 },
  'command-r-plus': { context: 128_000, maxOutput: 4_096 },
  'command-r-08-2024': { context: 128_000, maxOutput: 4_096 },
  'command-r-plus-08-2024': { context: 128_000, maxOutput: 4_096 },
  'command': { context: 4_096, maxOutput: 4_096 },
  'command-light': { context: 4_096, maxOutput: 4_096 },
  'command-light-nightly': { context: 4_096, maxOutput: 4_096 },
  'command-nightly': { context: 4_096, maxOutput: 4_096 },
  // xAI Grok
  'grok-3': { context: 131_072, maxOutput: 65_536 },
  'grok-3-mini': { context: 131_072, maxOutput: 65_536 },
  'grok-4': { context: 131_072, maxOutput: 65_536 },
  'grok-4-0709': { context: 131_072, maxOutput: 65_536 },
  'grok-4-fast-reasoning': { context: 131_072, maxOutput: 65_536 },
  'grok-4-fast-non-reasoning': { context: 131_072, maxOutput: 65_536 },
  'grok-4-1-fast-reasoning': { context: 131_072, maxOutput: 65_536 },
  'grok-4-1-fast-non-reasoning': { context: 131_072, maxOutput: 65_536 },
  // Tencent Hunyuan
  'hunyuan-lite': { context: 32_768, maxOutput: 8_192 },
  'hunyuan-standard': { context: 32_768, maxOutput: 8_192 },
  'hunyuan-standard-256k': { context: 262_144, maxOutput: 8_192 },
  'hunyuan-standard-256K': { context: 262_144, maxOutput: 8_192 },
  'hunyuan-pro': { context: 32_768, maxOutput: 8_192 },
  // Doubao
  'doubao-pro-128k': { context: 128_000, maxOutput: 8_192 },
  'doubao-pro-32k': { context: 32_768, maxOutput: 8_192 },
  'doubao-pro-4k': { context: 4_096, maxOutput: 4_096 },
  'doubao-lite-128k': { context: 128_000, maxOutput: 8_192 },
  'doubao-lite-32k': { context: 32_768, maxOutput: 8_192 },
  'doubao-lite-4k': { context: 4_096, maxOutput: 4_096 },
  'doubao-embedding': { context: 8_192, maxOutput: 0 },
  'doubao-seed-1-6-thinking-250715': { context: 128_000, maxOutput: 8_192 },
  // Baidu ERNIE
  'ernie-4.0-8k': { context: 8_192, maxOutput: 8_192 },
  'ernie-3.5-8k': { context: 8_192, maxOutput: 8_192 },
  'ernie-3.5-4k-0205': { context: 4_096, maxOutput: 4_096 },
  'ernie-speed-8k': { context: 8_192, maxOutput: 8_192 },
  'ernie-speed-128k': { context: 128_000, maxOutput: 8_192 },
  'ernie-lite-8k-0922': { context: 8_192, maxOutput: 8_192 },
  'ernie-lite-8k-0308': { context: 8_192, maxOutput: 8_192 },
  'ernie-tiny-8k': { context: 8_192, maxOutput: 8_192 },
  // Zhipu GLM
  'chatglm_turbo': { context: 8_192, maxOutput: 8_192 },
  'chatglm_pro': { context: 32_768, maxOutput: 8_192 },
  'chatglm_std': { context: 32_768, maxOutput: 8_192 },
  'chatglm_lite': { context: 8_192, maxOutput: 8_192 },
  // Gemini Gemma
  'gemma-3-1b-it': { context: 8_192, maxOutput: 8_192 },
  'gemma-3-4b-it': { context: 8_192, maxOutput: 8_192 },
  'gemma-3-12b-it': { context: 32_768, maxOutput: 32_768 },
  'gemma-3-27b-it': { context: 128_000, maxOutput: 32_768 },
  'gemma-3n-e4b-it': { context: 8_192, maxOutput: 8_192 },
  'gemma-3n-e2b-it': { context: 8_192, maxOutput: 8_192 },
  // Cloudflare Llama (approximate)
  '@cf/meta/llama-3.1-8b-instruct': { context: 128_000, maxOutput: 32_768 },
  '@cf/meta/llama-3-8b-instruct': { context: 8_192, maxOutput: 8_192 },
}

const TAG_TO_CAPABILITY: Record<string, ModelCapability> = {
  vision: 'vision',
  multimodal: 'vision',
  reasoning: 'reasoning',
  thinking: 'reasoning',
  tools: 'tools',
  function: 'function_calling',
  'function-calling': 'function_calling',
  streaming: 'streaming',
  json: 'json_mode',
  structured: 'structured_output',
  search: 'web_search',
  code: 'code_interpreter',
  embedding: 'embeddings',
}

const TAG_TO_MODALITY: Record<string, Modality> = {
  text: 'text',
  image: 'image',
  audio: 'audio',
  video: 'video',
  file: 'file',
  document: 'file',
  pdf: 'file',
}

function pickFromBuckets<T>(buckets: T[], rand: () => number): T {
  return buckets[Math.floor(rand() * buckets.length)]
}

function parseModelTags(tagsString?: string): string[] {
  if (!tagsString) return []
  return tagsString
    .split(/[,;|\s]+/)
    .map((t) => t.trim().toLowerCase())
    .filter(Boolean)
}

function nameMatches(name: string, patterns: RegExp[]): boolean {
  return patterns.some((re) => re.test(name))
}

function inferInputModalities(
  model: PricingModel,
  tags: string[],
  endpoints: string[],
  name: string
): Modality[] {
  const set = new Set<Modality>()

  if (
    endpoints.length === 0 ||
    endpoints.some((e) => TEXT_INPUT_ENDPOINTS.has(e))
  ) {
    set.add('text')
  }

  if (model.image_ratio != null || nameMatches(name, VISION_NAME_PATTERNS)) {
    set.add('image')
  }
  if (model.audio_ratio != null || nameMatches(name, AUDIO_NAME_PATTERNS)) {
    set.add('audio')
  }
  if (nameMatches(name, VIDEO_NAME_PATTERNS)) {
    set.add('video')
  }

  for (const tag of tags) {
    const m = TAG_TO_MODALITY[tag]
    if (m) set.add(m)
  }

  if (set.size === 0) set.add('text')
  return ordered(set)
}

function inferOutputModalities(
  model: PricingModel,
  endpoints: string[],
  name: string
): Modality[] {
  const set = new Set<Modality>()

  if (endpoints.some((e) => IMAGE_OUTPUT_ENDPOINTS.has(e))) set.add('image')
  if (endpoints.some((e) => VIDEO_OUTPUT_ENDPOINTS.has(e))) set.add('video')
  if (endpoints.some((e) => EMBEDDING_ENDPOINTS.has(e))) set.add('text')

  if (
    model.audio_completion_ratio != null ||
    /tts|voice|audio-out/i.test(name)
  ) {
    set.add('audio')
  }

  if (set.size === 0) set.add('text')
  return ordered(set)
}

function inferCapabilities(
  model: PricingModel,
  tags: string[],
  endpoints: string[],
  name: string,
  outputs: Modality[],
  inputs: Modality[]
): ModelCapability[] {
  const set = new Set<ModelCapability>()

  if (outputs.includes('text') && !endpoints.includes('image-generation')) {
    set.add('streaming')
    set.add('system_prompt')
  }
  if (
    !endpoints.includes('image-generation') &&
    !endpoints.includes('embeddings') &&
    !endpoints.includes('jina-rerank')
  ) {
    set.add('function_calling')
    set.add('tools')
    set.add('json_mode')
    set.add('structured_output')
  }
  if (inputs.includes('image')) set.add('vision')
  if (model.cache_ratio != null) set.add('caching')
  if (endpoints.some((e) => EMBEDDING_ENDPOINTS.has(e))) set.add('embeddings')
  if (nameMatches(name, REASONING_NAME_PATTERNS)) set.add('reasoning')
  if (nameMatches(name, CODE_NAME_PATTERNS)) set.add('code_interpreter')
  if (nameMatches(name, WEB_SEARCH_PATTERNS)) set.add('web_search')

  for (const tag of tags) {
    const cap = TAG_TO_CAPABILITY[tag]
    if (cap) set.add(cap)
  }

  return Array.from(set)
}

function ordered(modalities: Set<Modality>): Modality[] {
  const order: Modality[] = ['text', 'image', 'audio', 'video', 'file']
  return order.filter((m) => modalities.has(m))
}

function inferContextAndOutputs(
  name: string,
  rand: () => number,
  endpoints: string[]
): { context: number; maxOutput: number } {
  if (endpoints.includes('embeddings') || endpoints.includes('jina-rerank')) {
    return { context: 8_192, maxOutput: 0 }
  }
  if (
    endpoints.includes('image-generation') ||
    endpoints.includes('openai-video')
  ) {
    return { context: 4_096, maxOutput: 0 }
  }

  // 1. Exact match on known models
  const exact = KNOWN_MODEL_CONTEXT[name]
  if (exact) return exact

  const lower = name.toLowerCase()

  // 2. Regex patterns for model families (context from model name hints)
  if (lower.includes('1m') || lower.includes('-long')) {
    return { context: 1_000_000, maxOutput: 65_536 }
  }
  if (
    lower.includes('200k') ||
    lower.includes('claude-3') ||
    lower.includes('claude-4')
  ) {
    return { context: 200_000, maxOutput: 16_384 }
  }
  if (lower.includes('128k') || /gpt-4o|gpt-4\.1|gpt-5|o1|o3|o4/.test(lower)) {
    return { context: 128_000, maxOutput: 16_384 }
  }
  if (/gemini.*-2|gemini.*pro|gemini.*flash/.test(lower)) {
    return { context: 1_000_000, maxOutput: 8_192 }
  }
  if (/gpt-3\.5|claude-2/.test(lower)) {
    return { context: 16_384, maxOutput: 4_096 }
  }
  // DeepSeek v4
  if (/deepseek-v4/.test(lower)) {
    return { context: 64_000, maxOutput: 8_192 }
  }
  // DeepSeek v2/v3
  if (/deepseek/.test(lower)) {
    return { context: 64_000, maxOutput: 8_192 }
  }
  // Qwen
  if (/qwen|qwq/.test(lower)) {
    if (/longcontext|-1m/.test(lower)) {
      return { context: 1_000_000, maxOutput: 8_192 }
    }
    return { context: 128_000, maxOutput: 8_192 }
  }
  // Mistral
  if (/mistral|mixtral|codestral|magistral|pixtral/.test(lower)) {
    if (/large/.test(lower)) {
      return { context: 128_000, maxOutput: 32_768 }
    }
    return { context: 32_768, maxOutput: 8_192 }
  }
  // Cohere
  if (/command-r.*plus|r-plus/.test(lower)) {
    return { context: 128_000, maxOutput: 4_096 }
  }
  // xAI Grok
  if (/grok-?[34]/.test(lower)) {
    return { context: 131_072, maxOutput: 65_536 }
  }
  // Kimi
  if (/kimi/.test(lower)) {
    return { context: 128_000, maxOutput: 32_768 }
  }
  // MiniMax
  if (/minimax|abab/i.test(lower)) {
    return { context: 32_768, maxOutput: 8_192 }
  }
  // Hunyuan
  if (/hunyuan/.test(lower)) {
    if (/256k/i.test(lower)) {
      return { context: 262_144, maxOutput: 8_192 }
    }
    return { context: 32_768, maxOutput: 8_192 }
  }
  // Doubao
  if (/doubao/.test(lower)) {
    if (/-128k/i.test(lower)) return { context: 128_000, maxOutput: 8_192 }
    if (/-32k/i.test(lower)) return { context: 32_768, maxOutput: 8_192 }
    return { context: 32_768, maxOutput: 8_192 }
  }
  // ERNIE
  if (/ernie/.test(lower)) {
    if (/-128k/i.test(lower)) return { context: 128_000, maxOutput: 8_192 }
    return { context: 8_192, maxOutput: 8_192 }
  }
  // GLM / Zhipu
  if (/chatglm|glm/.test(lower)) {
    return { context: 32_768, maxOutput: 8_192 }
  }
  // Gemma
  if (/gemma-3-27b/.test(lower)) {
    return { context: 128_000, maxOutput: 32_768 }
  }
  if (/gemma-3-12b/.test(lower)) {
    return { context: 32_768, maxOutput: 32_768 }
  }
  if (/gemma/.test(lower)) {
    return { context: 8_192, maxOutput: 8_192 }
  }
  // Llama on Cloudflare
  if (/llama.*3\.1.*8b/.test(lower)) {
    return { context: 128_000, maxOutput: 32_768 }
  }
  // GPT-4 (non-turbo)
  if (/^gpt-4$|^gpt-4-\d{4}/.test(lower) && !/turbo|4o|4\.1|4\.5/.test(lower)) {
    return { context: 8_192, maxOutput: 8_192 }
  }
  // GPT-4-turbo
  if (/gpt-4-turbo/.test(lower)) {
    return { context: 128_000, maxOutput: 4_096 }
  }
  // GPT-4-32k
  if (/gpt-4-32k/.test(lower)) {
    return { context: 32_768, maxOutput: 32_768 }
  }

  // 3. Fallback: random bucket (deterministic by model name)
  const context = pickFromBuckets(CONTEXT_BUCKETS, rand)
  const maxOutput = Math.min(context, pickFromBuckets(MAX_OUTPUT_BUCKETS, rand))
  return { context, maxOutput }
}

function inferReleaseAndCutoff(rand: () => number): {
  release: string
  cutoff: string
} {
  const cutoff = pickFromBuckets(KNOWLEDGE_CUTOFFS, rand)
  const [year, month] = cutoff.split('-').map(Number)
  const offsetMonths = 4 + Math.floor(rand() * 6)
  const releaseMonth = month + offsetMonths
  const releaseYear = year + Math.floor((releaseMonth - 1) / 12)
  const finalMonth = ((releaseMonth - 1) % 12) + 1
  const release = `${releaseYear}-${String(finalMonth).padStart(2, '0')}-15`
  return { release, cutoff }
}

export type ModelMetadata = {
  context_length: number
  max_output_tokens: number
  knowledge_cutoff: string
  release_date: string
  parameter_count: string
  input_modalities: Modality[]
  output_modalities: Modality[]
  capabilities: ModelCapability[]
}

/**
 * Infer / mock model metadata. Prefers explicit fields on `model.*` and
 * falls back to inference + a deterministic seed otherwise.
 */
export function inferModelMetadata(model: PricingModel): ModelMetadata {
  const name = model.model_name || ''
  const rand = seededRandom(hashStringToSeed(name))
  const tags = parseModelTags(model.tags)
  const endpoints = model.supported_endpoint_types || []

  const inputs =
    model.input_modalities ?? inferInputModalities(model, tags, endpoints, name)
  const outputs =
    model.output_modalities ?? inferOutputModalities(model, endpoints, name)
  const capabilities =
    model.capabilities ??
    inferCapabilities(model, tags, endpoints, name, outputs, inputs)

  const fallback = inferContextAndOutputs(name, rand, endpoints)
  const cutoffAndRelease = inferReleaseAndCutoff(rand)

  return {
    context_length: model.context_length ?? fallback.context,
    max_output_tokens: model.max_output_tokens ?? fallback.maxOutput,
    knowledge_cutoff: model.knowledge_cutoff ?? cutoffAndRelease.cutoff,
    release_date: model.release_date ?? cutoffAndRelease.release,
    parameter_count:
      model.parameter_count ?? pickFromBuckets(PARAM_BUCKETS, rand),
    input_modalities: inputs,
    output_modalities: outputs,
    capabilities,
  }
}

const TOKEN_FORMAT = new Intl.NumberFormat(undefined, {
  maximumFractionDigits: 1,
})

/** Format a token count compactly: 128_000 → "128K", 1_000_000 → "1M". */
export function formatTokenCount(tokens: number): string {
  if (!Number.isFinite(tokens) || tokens <= 0) return '—'
  if (tokens >= 1_000_000) {
    const value = tokens / 1_000_000
    return `${TOKEN_FORMAT.format(value)}M`
  }
  if (tokens >= 1_000) {
    const value = tokens / 1_000
    return `${TOKEN_FORMAT.format(value)}K`
  }
  return TOKEN_FORMAT.format(tokens)
}

/** Format a YYYY-MM (or YYYY-MM-DD) date as `Mon YYYY` for display. */
export function formatYearMonth(value: string): string {
  if (!value) return '—'
  const [yearStr, monthStr] = value.split('-')
  const year = Number(yearStr)
  const month = Number(monthStr)
  if (!Number.isFinite(year) || !Number.isFinite(month)) return value
  const date = new Date(Date.UTC(year, month - 1, 1))
  return date.toLocaleString(undefined, { year: 'numeric', month: 'short' })
}

// ---------------------------------------------------------------------------
// Provider / vendor / tokenizer / license inference
// ---------------------------------------------------------------------------
//
// These helpers derive vendor-style metadata from the model name. They are
// purely heuristic and serve only the API-info display until the backend
// returns explicit fields.

export type ModelVendor =
  | 'openai'
  | 'anthropic'
  | 'google'
  | 'meta'
  | 'mistral'
  | 'qwen'
  | 'deepseek'
  | 'xai'
  | 'cohere'
  | 'baidu'
  | 'zhipu'
  | 'moonshot'
  | 'minimax'
  | 'tencent'
  | 'bytedance'
  | 'midjourney'
  | 'stability'
  | 'unknown'

export type ApiInfo = {
  vendor: ModelVendor
  vendor_label: string
  tokenizer: string
  tokenizer_note?: string
  license: string
  license_kind: 'proprietary' | 'open' | 'open-weight' | 'unknown'
  data_retention_days: number
  training_opt_out: boolean
  homepage?: string
}

const VENDOR_LABELS: Record<ModelVendor, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  google: 'Google',
  meta: 'Meta',
  mistral: 'Mistral AI',
  qwen: 'Alibaba (Qwen)',
  deepseek: 'DeepSeek',
  xai: 'xAI',
  cohere: 'Cohere',
  baidu: 'Baidu',
  zhipu: 'Zhipu AI',
  moonshot: 'Moonshot AI',
  minimax: 'MiniMax',
  tencent: 'Tencent',
  bytedance: 'ByteDance',
  midjourney: 'Midjourney',
  stability: 'Stability AI',
  unknown: 'Unknown',
}

function detectVendor(name: string): ModelVendor {
  const n = name.toLowerCase()
  if (/^gpt|^o[1-4]|davinci|babbage|whisper|tts|dall.?e|sora|^omni/.test(n))
    return 'openai'
  if (/claude/.test(n)) return 'anthropic'
  if (/gemini|gemma|imagen|veo|palm/.test(n)) return 'google'
  if (/llama|^codellama/.test(n)) return 'meta'
  if (/mistral|mixtral|codestral|magistral|pixtral/.test(n)) return 'mistral'
  if (/qwen|qwq|qvq/.test(n)) return 'qwen'
  if (/deepseek/.test(n)) return 'deepseek'
  if (/grok/.test(n)) return 'xai'
  if (/command|cohere|aya/.test(n)) return 'cohere'
  if (/ernie|wenxin/.test(n)) return 'baidu'
  if (/glm|chatglm|cogview|cogvideo/.test(n)) return 'zhipu'
  if (/kimi|moonshot/.test(n)) return 'moonshot'
  if (/abab|minimax|hailuo/.test(n)) return 'minimax'
  if (/hunyuan/.test(n)) return 'tencent'
  if (/doubao|seed|jimeng/.test(n)) return 'bytedance'
  if (/midjourney|niji/.test(n)) return 'midjourney'
  if (/^sd-|stable[-_]?diffusion|sdxl/.test(n)) return 'stability'
  return 'unknown'
}

const TOKENIZER_BY_VENDOR: Partial<Record<ModelVendor, string>> = {
  openai: 'o200k_base',
  anthropic: 'Anthropic Claude tokenizer',
  google: 'SentencePiece (Gemini)',
  meta: 'Llama 3 tokenizer',
  mistral: 'Mistral tokenizer (BPE)',
  qwen: 'Qwen tokenizer (tiktoken-compat)',
  deepseek: 'DeepSeek tokenizer (BPE)',
  xai: 'Grok tokenizer (BPE)',
  cohere: 'Cohere tokenizer',
  baidu: 'Ernie tokenizer',
  zhipu: 'GLM tokenizer',
  moonshot: 'Kimi tokenizer',
  minimax: 'ABAB tokenizer',
  tencent: 'Hunyuan tokenizer',
  bytedance: 'Doubao tokenizer',
}

function inferTokenizer(
  model: PricingModel,
  vendor: ModelVendor
): {
  tokenizer: string
  note?: string
} {
  const name = model.model_name.toLowerCase()
  if (vendor === 'openai') {
    if (/gpt-3|davinci|babbage|whisper|tts/.test(name)) {
      return { tokenizer: 'cl100k_base', note: 'Older GPT-3.5 family' }
    }
    return { tokenizer: 'o200k_base' }
  }
  return { tokenizer: TOKENIZER_BY_VENDOR[vendor] ?? 'BPE (vendor-specific)' }
}

const LICENSE_BY_VENDOR: Record<
  ModelVendor,
  { license: string; kind: ApiInfo['license_kind'] }
> = {
  openai: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  anthropic: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  google: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  meta: { license: 'Llama Community License', kind: 'open-weight' },
  mistral: { license: 'Apache 2.0 / Commercial', kind: 'open-weight' },
  qwen: { license: 'Tongyi Qianwen License', kind: 'open-weight' },
  deepseek: { license: 'DeepSeek License', kind: 'open-weight' },
  xai: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  cohere: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  baidu: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  zhipu: { license: 'GLM-4 License', kind: 'open-weight' },
  moonshot: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  minimax: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  tencent: { license: 'Hunyuan License', kind: 'open-weight' },
  bytedance: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  midjourney: { license: 'Proprietary (commercial)', kind: 'proprietary' },
  stability: { license: 'Stability AI Community License', kind: 'open-weight' },
  unknown: { license: 'Provider-specific', kind: 'unknown' },
}

const HOMEPAGE_BY_VENDOR: Partial<Record<ModelVendor, string>> = {
  openai: 'https://platform.openai.com/docs/models',
  anthropic: 'https://docs.anthropic.com/claude/docs/models-overview',
  google: 'https://ai.google.dev/models',
  meta: 'https://llama.meta.com/',
  mistral: 'https://docs.mistral.ai/getting-started/models/',
  qwen: 'https://qwenlm.github.io/',
  deepseek: 'https://api-docs.deepseek.com/',
  xai: 'https://x.ai/api',
  cohere: 'https://docs.cohere.com/docs/models',
  baidu: 'https://cloud.baidu.com/product/wenxinworkshop',
  zhipu: 'https://open.bigmodel.cn/dev/api',
  moonshot: 'https://platform.moonshot.cn/docs',
  minimax: 'https://platform.minimaxi.com/document/notice',
  tencent: 'https://cloud.tencent.com/document/product/1729',
  bytedance: 'https://www.volcengine.com/docs/82379',
  midjourney: 'https://www.midjourney.com/',
  stability: 'https://platform.stability.ai/',
}

/**
 * Build vendor / tokenizer / license / privacy metadata for the model.
 * Returns deterministic values keyed off the model name so each render is
 * stable.
 */
export function inferApiInfo(model: PricingModel): ApiInfo {
  const vendor = detectVendor(model.model_name || '')
  const tk = inferTokenizer(model, vendor)
  const license = LICENSE_BY_VENDOR[vendor]
  const rand = seededRandom(hashStringToSeed(`${model.model_name}:api`))
  const retention = vendor === 'openai' ? 30 : Math.round(rand() * 90)
  return {
    vendor,
    vendor_label: VENDOR_LABELS[vendor],
    tokenizer: tk.tokenizer,
    tokenizer_note: tk.note,
    license: license.license,
    license_kind: license.kind,
    data_retention_days: retention,
    training_opt_out: true,
    homepage: HOMEPAGE_BY_VENDOR[vendor],
  }
}
