# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Navigation

### 📋 Core Documentation
- **[System Architecture](docs/architecture/ARCHITECTURE.md)** - Project overview, key components, and Clean Architecture layers
- **[Architectural Patterns](docs/architecture/PATTERNS.md)** - Important design patterns and best practices
- **[API Reference](docs/architecture/API_REFERENCE.md)** - HTTP endpoints and API documentation

### 🛠️ Development
- **[Development Commands](docs/development/COMMANDS.md)** - Common commands for building, testing, and development
- **[Development Workflow](docs/development/WORKFLOW.md)** - Guidelines for adding new features and testing strategies

### 🚀 Deployment & Configuration
- **[Configuration Management](docs/deployment/CONFIGURATION.md)** - Configuration setup and environment variables
- **[Event Flow](docs/deployment/EVENT_FLOW.md)** - System event flow and KDS integration

### 📈 Project Status & Updates
- **[Current Status](docs/updates/CURRENT_STATUS.md)** - Latest version status and enterprise-grade achievements
- **[Recent Updates](docs/updates/RECENT_UPDATES.md)** - Major recent architecture updates and optimizations
- **[Archive Reference](docs/updates/ARCHIVE_REFERENCE.md)** - Complete development history and archived documentation

## Project Summary

Fat Identity Cat is a Go-based microservice for managing merchant, player, and manager identities with synchronization capabilities through AWS Kinesis Data Streams (KDS). The system follows Clean Architecture principles with hexagonal design and includes three main services:

1. **Web Service** - HTTP API server with REST endpoints
2. **Consumer Service** - KDS event processing with high-performance batch optimization
3. **Worker Service** - Background job processing with Redis queue integration

## Current Architecture Status

✅ **Enterprise-Grade Performance**: 500x reduction in database calls, sub-millisecond cache lookups, 99.95%+ stability  
✅ **Production Deployed**: All optimizations running in production with 3.3x performance improvement  
✅ **Complete Type Safety**: Universal cache integration with compile-time type checking  
✅ **Zero Data Loss**: Comprehensive batch processing with fallback mechanisms  

## Important Instruction Reminders

Do what has been asked; nothing more, nothing less.  
NEVER create files unless they're absolutely necessary for achieving your goal.  
ALWAYS prefer editing an existing file to creating a new one.  
NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.