# Custom Android Skills

This directory contains specialized Claude Code skills for Android development and forensic analysis.

## Available Skills

### 1. **android-forensic**

Expert forensic analysis of Android applications with defensive and ethical approach.

**Activates when you ask about:**

- APK/DEX analysis
- Frida instrumentation
- Root detection bypass
- Certificate pinning bypass
- Crash analysis
- Security auditing
- Reverse engineering

### 2. **android-senior-dev**

Senior/Staff-level Android development guidance for production-ready applications.

**Activates when you ask about:**

- Architecture patterns (MVVM, MVI, Clean Architecture)
- Jetpack Compose implementation
- Kotlin coroutines and Flow
- Performance optimization
- Security hardening
- Testing strategies
- Production deployment

## Skill Structure

Each skill follows this organization:

```
skill-name/
├── SKILL.md              # Main skill definition (REQUIRED)
├── reference.md          # Extended reference documentation
├── examples.md           # Code examples and patterns
├── scripts/              # Utility scripts
│   ├── *.sh             # Shell scripts
│   └── *.js             # Frida/JavaScript scripts
└── README.md            # Skill-specific documentation
```

## How to Extend Skills

### Option 1: Add Reference Documentation

Create additional markdown files for detailed references:

```bash
touch skills/android-forensic/advanced-topics.md
```

Reference other files in `SKILL.md`:

```markdown
See [advanced-topics.md](advanced-topics.md) for more details.
```

### Option 2: Add Code Examples

Create `examples.md` with complete code samples:

```markdown
# Example: Implement Feature X

## Step 1: Domain Layer
\`\`\`kotlin
// Your code here
\`\`\`

## Step 2: Data Layer
\`\`\`kotlin
// Your code here
\`\`\`
```

### Option 3: Add Utility Scripts

Create scripts that can be referenced in skills:

```bash
# Create script directory
mkdir -p skills/android-forensic/scripts

# Add a script
cat > skills/android-forensic/scripts/my-tool.sh << 'EOF'
#!/bin/bash
# Your script here
EOF

chmod +x skills/android-forensic/scripts/my-tool.sh
```

### Option 4: Update SKILL.md

Add new sections to the main skill file:

```markdown
## New Section

### Subsection 1
Content here...

### Subsection 2
More content...
```

### Option 5: Add Checklists

Create interactive checklists:

```markdown
## Pre-Release Checklist

### Security
- [ ] Certificate pinning enabled
- [ ] ProGuard rules updated
- [ ] No hardcoded secrets

### Performance
- [ ] Startup time < 2s
- [ ] No memory leaks
- [ ] ANR rate < 0.5%
```

### Option 6: Add Tool Configurations

Store configuration files for tools:

```bash
# ProGuard rules template
skills/android-senior-dev/configs/proguard-template.pro

# Detekt configuration
skills/android-senior-dev/configs/detekt.yml

# Frida script template
skills/android-forensic/configs/frida-template.js
```

## Updating Skill Descriptions

The `description` field in the YAML frontmatter is crucial for skill discovery:

```yaml
---
name: my-skill
description: What it does (triggers) + when to use it (use cases)
---
```

**Good descriptions:**

```yaml
description: Analyze Android crash dumps, ANRs, and tombstones. Use for debugging production crashes, investigating memory issues, or analyzing native crashes.
```

**Bad descriptions:**

```yaml
description: A skill for crashes
```

## Skill Best Practices

### 1. Keep SKILL.md Focused

- Core concepts and patterns
- Common use cases
- Quick reference

### 2. Use Supporting Files for Detail

- `reference.md` for comprehensive documentation
- `examples.md` for complete implementations
- `scripts/` for automation tools

### 3. Include Practical Examples

```kotlin
// BAD: Abstract explanation
"Use dependency injection"

// GOOD: Concrete example
@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {
    @Provides
    @Singleton
    fun provideOkHttpClient(): OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .build()
}
```

### 4. Add Troubleshooting Sections

```markdown
## Common Issues

### Issue: Build fails with "Duplicate class"
**Solution:** Add to build.gradle.kts:
\`\`\`kotlin
configurations.all {
    exclude(group = "org.jetbrains.kotlin", module = "kotlin-stdlib-jdk7")
}
\`\`\`
```

### 5. Version-Specific Notes

```markdown
## Version Compatibility

### Android 14 (API 34+)
- New photo picker API
- Predictive back gesture

### Android 13 (API 33+)
- Runtime notification permission
- Photo picker
```

## Testing Your Skills

After creating/updating a skill:

1. **Test activation:**
   ```
   Ask a question that should trigger the skill
   ```

2. **Verify file references:**
   ```
   Check that linked files exist and are readable
   ```

3. **Review responses:**
   ```
   Ensure responses match your expectations
   ```

## Sharing Skills with Team

Skills in `skills/` are project-specific:

```bash
# Commit to git
git add skills/
git commit -m "Add Android forensic and senior dev skills"
git push

# Team members get skills on next pull
git pull
```

## Example: Creating a New Skill

```bash
# 1. Create skill directory
mkdir -p skills/android-testing

# 2. Create SKILL.md
cat > skills/android-testing/SKILL.md << 'EOF'
---
name: android-testing
description: Comprehensive Android testing strategies including unit tests, UI tests, integration tests, and test automation. Use for writing tests, debugging test failures, or setting up CI/CD testing pipelines.
---

# Android Testing Expert

**IMPORTANT: Always respond in English, regardless of the language used in the question.**

## Testing Strategy

### Unit Tests
...

### UI Tests
...
EOF

# 3. Add examples
cat > skills/android-testing/examples.md << 'EOF'
# Testing Examples

## Unit Test Example
...
EOF

# 4. Test it
# Ask: "How do I write unit tests for my ViewModel?"
```

## Resources

- [Claude Code Documentation](https://claude.ai/code)
- [Android Developer Guide](https://developer.android.com)
- [Kotlin Documentation](https://kotlinlang.org/docs/)
