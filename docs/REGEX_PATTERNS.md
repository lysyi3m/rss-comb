# Regex Pattern Support

RSS Comb supports both **substring matching** (existing) and **regular expressions** (new) in filter patterns, allowing for more powerful and concise feed filtering.

## Pattern Syntax

### Substring Patterns (existing behavior)
```yaml
excludes:
  - "weekly digest"    # Matches any title containing this substring
  - "advertisement"    # Case-insensitive
```

### Regex Patterns (new)
```yaml
excludes:
  - "/^mobile development/"  # Matches titles starting with this text
  - "/weekly|digest/"        # Matches either "weekly" OR "digest"
  - "/\\d+ events/"          # Matches "5 events", "10 events", etc.
```

**Convention**: Patterns wrapped in `/slashes/` are treated as regular expressions.

## Key Features

- ✅ **Case-insensitive by default** - All regex patterns automatically use `(?i)` flag
- ✅ **Mixed patterns** - Use both regex and substring patterns together
- ✅ **Unicode normalization** - Applied before regex matching
- ✅ **Cached compilation** - Regex patterns compiled once and cached for performance
- ✅ **Graceful fallback** - Invalid regex patterns fall back to literal substring matching with warning

## Practical Examples

### Before: Multiple Similar Patterns
```yaml
filters:
  - field: "title"
    excludes:
      - "Mobile development weekly"
      - "Security news weekly"
      - "TOP-5 events of the week"
      - "Weekly digest"
```

### After: Single Regex Pattern
```yaml
filters:
  - field: "title"
    excludes:
      - "/weekly|week/"  # Matches all weekly-related content
```

### Exclude Link Patterns
```yaml
filters:
  - field: "link"
    excludes:
      - "/\\/companies\\/(acme|widgets|corp)\\/"  # Multiple companies
```

### Category Filtering
```yaml
filters:
  - field: "categories"
    excludes:
      - "/^(angular|vue|react)$/"  # Exact match only
      - "/^top-?\\d+/"              # TOP-5, top10, etc.
```

### Include Patterns
```yaml
filters:
  - field: "title"
    includes:
      - "/^tech(nology)?/"         # Starts with "tech" or "technology"
      - "machine learning"          # Simple substring (mixed with regex)
```

## Common Regex Patterns

| Pattern | Matches | Example |
|---------|---------|---------|
| `/^word/` | Starts with "word" | "Word of the day" ✓, "The word" ✗ |
| `/word$/` | Ends with "word" | "The final word" ✓, "Word is" ✗ |
| `/word1\|word2/` | Either word1 or word2 | "word1", "word2" ✓ |
| `/\\d+/` | Contains any number | "5 items", "123" ✓ |
| `/^(a\|b\|c)$/` | Exactly a, b, or c | "a", "b", "c" ✓, "ab" ✗ |
| `/.*/` | Matches everything | Use with caution |

## Performance

- **First match**: Pattern compiled and cached (~1ms)
- **Subsequent matches**: Retrieved from cache (~0.001ms)
- **Cache lifetime**: Until config reload or app restart
- **Memory overhead**: ~1-2KB per unique regex pattern

## Cache Behavior

The regex compilation cache:
- Persists for application lifetime
- Automatically cleared on config reload (`/api/feeds/:name/reload`)
- Thread-safe (uses `sync.Map`)
- No size limit (assumes stable feed configs)

## Debugging Invalid Patterns

If a regex pattern is invalid, the system:
1. Logs a warning: `Invalid regex pattern "/[invalid/": error parsing regexp...`
2. Falls back to literal substring matching
3. Continues processing (doesn't break the feed)

Check application logs for regex compilation errors.

## Escaping Special Characters

Regex special characters need escaping with `\\`:
- Dot: `\\.` (matches literal dot)
- Slash: `\\/` (matches literal slash)
- Brackets: `\\[`, `\\]`
- Parentheses: `\\(`, `\\)`
- Pipe: `\\|` (for literal pipe, not OR)

## Complete Real-World Example

```yaml
url: "https://example.com/feed.xml"
enabled: true

settings:
  refresh_interval: 1800
  max_items: 50

filters:
  # Exclude weekly digests (regex)
  - field: "title"
    excludes:
      - "/weekly|week|digest/"

  # Exclude specific companies (mixed)
  - field: "link"
    excludes:
      - "/\\/companies\\/(acme|widgets|corp)\\/"
      - "/promo/"        # Regex
      - "?utm_source"    # Substring

  # Include specific categories (regex + substring)
  - field: "categories"
    includes:
      - "/^(go|rust|python)$/"  # Exact programming language match
      - "machine learning"       # Substring match
    excludes:
      - "/^top-?\\d+/"           # Exclude TOP-N lists
```
