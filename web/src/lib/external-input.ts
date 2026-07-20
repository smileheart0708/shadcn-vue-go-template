import type { z } from 'zod'

export function parseExternalValue<TSchema extends z.ZodType>(
  schema: TSchema,
  value: unknown,
): z.infer<TSchema> | null {
  const parsedValue = schema.safeParse(value)
  return parsedValue.success ? parsedValue.data : null
}

export function parseExternalValueOrDefault<TSchema extends z.ZodType>(
  schema: TSchema,
  value: unknown,
  fallbackValue: z.infer<TSchema>,
): z.infer<TSchema> {
  return parseExternalValue(schema, value) ?? fallbackValue
}

export function parseExternalJsonOrDefault<TSchema extends z.ZodType>(
  schema: TSchema,
  rawValue: string | null,
  fallbackValue: z.infer<TSchema>,
): z.infer<TSchema> {
  if (rawValue === null) {
    return fallbackValue
  }

  try {
    const parsedJson: unknown = JSON.parse(rawValue)
    return parseExternalValueOrDefault(schema, parsedJson, fallbackValue)
  } catch {
    return fallbackValue
  }
}
