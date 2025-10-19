# Keyline Onboarding Guide

Welcome to Keyline! This comprehensive onboarding guide will help you understand Keyline's architecture, design patterns, and development workflows.

## 📚 Table of Contents

1. **[Architecture Overview](01-architecture-overview.md)** - Understand the big picture
2. **[CQRS and Mediator Pattern](02-cqrs-and-mediator.md)** - Learn the core communication patterns
3. **[Dependency Injection with IoC](03-dependency-injection.md)** - Understand how components are wired together
4. **[Development Workflow](04-development-workflow.md)** - Get started with development
5. **[Testing Guide](06-testing-guide.md)** - Write effective tests

## 🎯 Quick Start Path

### For Contributors New to Keyline

**Estimated Time: 2 hours**

1. Start with the [Architecture Overview](01-architecture-overview.md) (30 min)
   - Understand clean architecture principles
   - Learn the folder structure
   - See how components interact

2. Deep dive into [CQRS and Mediator](02-cqrs-and-mediator.md) (45 min)
   - Understand commands vs queries
   - Learn the mediator pattern
   - See how requests flow through the system

3. Explore [Dependency Injection](03-dependency-injection.md) (30 min)
   - Learn about IoC container
   - Understand lifetime management
   - See how dependencies are resolved

4. Follow the [Development Workflow](04-development-workflow.md) (15 min)
   - Set up your environment
   - Make your first change
   - Run tests and linting

### For Quick Reference

Already familiar with the basics? Jump to specific sections:

- **Writing tests?** → See [Testing Guide](06-testing-guide.md)
- **Understanding errors?** → See [Development Workflow: Troubleshooting](04-development-workflow.md#troubleshooting)
- **Need examples?** → Check existing commands and queries in `internal/commands/` and `internal/queries/`

## 🏗️ Architecture at a Glance

Keyline follows **Clean Architecture** with **CQRS** (Command Query Responsibility Segregation) pattern:

```
┌─────────────────────────────────────────────────────────────┐
│                        HTTP Layer                            │
│                    (Handlers/Routes)                         │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    Mediator Layer                            │
│              (Commands/Queries + Behaviors)                  │
└──────────────────────┬──────────────────────────────────────┘
                       │
          ┌────────────┴────────────┐
          ▼                         ▼
┌──────────────────┐       ┌──────────────────┐
│   Command Layer  │       │   Query Layer    │
│   (Write Logic)  │       │   (Read Logic)   │
└────────┬─────────┘       └────────┬─────────┘
         │                          │
         └──────────┬───────────────┘
                    ▼
┌─────────────────────────────────────────────────────────────┐
│                   Repository Layer                           │
│                  (Data Access Logic)                         │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                      Database                                │
│               (PostgreSQL/SQLite)                            │
└─────────────────────────────────────────────────────────────┘
```

## 🔑 Key Principles

### 1. **Separation of Concerns**
Each layer has a clear responsibility. Handlers don't access databases directly; they use commands and queries through the mediator.

### 2. **Dependency Inversion**
High-level modules don't depend on low-level modules. Both depend on abstractions (interfaces).

### 3. **Single Responsibility**
Each component does one thing well. Commands modify state, queries read data, handlers handle HTTP.

### 4. **Decoupled Communication**
Components communicate through the mediator, not directly. This makes the system flexible and testable.

## 🎓 Learning Resources

### Internal Documentation
- [IoC Container Documentation](../../ioc/Readme.md) - Deep dive into dependency injection
- [Mediator Pattern Documentation](../../mediator/README.md) - Deep dive into the mediator
- [Configuration Documentation](../../internal/config/README.md) - Configuration management
- [E2E Testing Documentation](../../tests/e2e/README.md) - End-to-end testing guide
- [API Client Documentation](../../client/README.md) - Using the API client

### External Resources
- [Clean Architecture by Robert Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
- [Go Project Layout](https://github.com/golang-standards/project-layout)

## 🤝 Getting Help

- **GitHub Issues**: [Report bugs or ask questions](https://github.com/The127/Keyline/issues)
- **GitHub Discussions**: [Community discussions](https://github.com/The127/Keyline/discussions)
- **Code Comments**: Many complex sections have detailed comments explaining the "why"

## 📝 Contributing

After reading this guide, you'll be ready to contribute! Check the main [README](../../README.md) for:
- Code style guidelines
- Testing requirements
- Pull request process
- Developer Certificate of Origin (DCO)

---

**Ready to dive in?** Start with the [Architecture Overview](01-architecture-overview.md) →
