# Backend Architecture & Design Principles

This document outlines the software engineering principles, architectural patterns, design patterns, and Object-Oriented concepts applied in the development of the Go backend for the Task Manager Application.

## 1. Architectural Patterns

### RESTful API Architecture
The backend is built following Representational State Transfer (REST) conventions. It exposes stateless, cacheable endpoints (`/api/tasks`, `/api/auth`) where HTTP methods (`GET`, `POST`, `PUT`, `DELETE`) clearly define the operations to be performed on the resources (Tasks and Users).

### Client-Server Architecture
The system employs a strict separation of concerns between the React frontend (Client) and the Go backend (Server). They communicate exclusively over a network boundary using JSON payloads, allowing both to evolve independently.

### Middleware Architecture (Pipeline)
The application leverages a middleware pipeline architecture to intercept HTTP requests.
*   **Authentication Pipeline:** The `authMiddleware` intercepts requests to protected routes, validates the JWT, and enriches the request header with the `X-User-ID` before passing control to the final handler.
*   **CORS Pipeline:** The `corsMiddleware` sits at the very edge of the server, appending necessary Access-Control headers to every response to satisfy browser security policies.

## 2. Design Patterns

### Middleware / Decorator Pattern
In Go, middleware is a classic implementation of the Decorator pattern. Functions like `corsMiddleware(next http.Handler)` dynamically wrap the core business logic with additional behavior (like attaching headers) without altering the original handler's code.

### Singleton Pattern (Connection Pooling)
The `db *sql.DB` variable acts as a Singleton. It is initialized once during the application startup (`initDB()`) and provides a thread-safe connection pool that is shared across all incoming HTTP requests throughout the lifecycle of the application.

### Data Transfer Object (DTO) Pattern
To separate database entities from network payloads, the backend utilizes DTOs such as `UserRequest` and `TaskRequest`. This prevents accidental over-posting and allows the API shape to differ from the database schema when necessary.

## 3. SOLID Principles

Although Go is not a traditional class-based language, SOLID principles heavily influence its idiomatic design:

*   **Single Responsibility Principle (SRP):** 
    Each handler function has one distinct job. For instance, `loginHandler` is solely responsible for authenticating a user and issuing a token. It delegates database connection management to `sql.DB` and token creation to `generateToken()`.
*   **Open/Closed Principle (OCP):**
    The routing mechanism (`http.ServeMux`) is open for extension but closed for modification. We can seamlessly add new API routes without changing the logic of existing routes or the core server configuration.
*   **Liskov Substitution Principle (LSP):**
    Any function conforming to the signature `func(http.ResponseWriter, *http.Request)` can be cast to an `http.HandlerFunc` and substituted wherever an `http.Handler` is expected (like in our middleware wrappers).
*   **Interface Segregation Principle (ISP):**
    Go standard library interfaces used here are purposefully small. `http.ResponseWriter` only contains the methods necessary to respond to a client (`Write`, `WriteHeader`, `Header`), rather than a bloated, monolithic interface.
*   **Dependency Inversion Principle (DIP):**
    Handlers rely on abstractions (`http.ResponseWriter`, `*http.Request`) rather than concrete network socket implementations, decoupling the HTTP application logic from the underlying TCP networking.

## 4. Object-Oriented Programming (OOP) Concepts in Go

Go implements OOP differently than Java or C#, strongly favoring composition over inheritance:

*   **Encapsulation:** 
    Data hiding is achieved through Go's capitalization rules. Additionally, we encapsulate sensitive data using struct tags. For instance, `json:"-"` on the `Password` field in the `AppUser` struct ensures passwords are mathematically guaranteed to never leak into JSON API responses.
*   **Composition over Inheritance:**
    Instead of extending a base `Entity` class, Go models (`Task`, `AppUser`) are plain structs. If shared behavior is needed in the future, Go uses struct embedding to compose behavior rather than inheriting it.
*   **Polymorphism:**
    Through the use of interfaces (like the `http.Handler` interface returned by the CORS middleware), different types can be treated uniformly by the `http.ListenAndServe` function.

## 5. Other Key Technical Highlights

*   **Implicit Concurrency:** 
    Unlike NodeJS which is single-threaded, Go's `net/http` package automatically spawns a lightweight **goroutine** for every incoming HTTP request. This means `createTaskHandler` and `getTasksHandler` can execute concurrently in parallel for thousands of users without manual thread management.
*   **Stateless Security (JWT):** 
    Authentication relies on JSON Web Tokens mapped using HMAC SHA-256 (`HS256`). This means the server does not need to store active sessions in memory, allowing the backend to scale horizontally effortlessly.
*   **Cryptographic Salting & Hashing:**
    Passwords are never stored in plaintext. The backend utilizes `bcrypt.GenerateFromPassword`, which automatically generates a unique cryptographic salt for every user and hashes the password safely, migrating the exact security standard from the previous Spring Boot backend.
