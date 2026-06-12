# Архітектура та UML

CourseHub — система керування курсами освітнього закладу: реєстрація студентів,
каталог курсів, зарахування та оцінювання з відстеженням прогресу. Код
структуровано за принципом чистої багатошарової архітектури з інверсією
залежностей (залежності спрямовані всередину, до `domain`).

## Шари

```mermaid
flowchart TD
    subgraph Presentation
        H["httpapi<br/>REST + web UI"]
    end
    subgraph Application
        S["service<br/>бізнес-логіка"]
    end
    subgraph Domain
        D["domain<br/>сутності + помилки"]
        G["grading<br/>Strategy"]
        P["progress<br/>Observer"]
        C["course<br/>Factory + Decorator"]
    end
    subgraph Infrastructure
        ST["storage<br/>SQLite Repository"]
    end
    CMD["cmd/server<br/>composition root"]

    H --> S
    S --> G
    S --> P
    S --> C
    S --> D
    C --> G
    C --> D
    P --> G
    ST -. реалізує інтерфейси .-> S
    CMD --> H
    CMD --> S
    CMD --> ST
    CMD --> P
```

Ключова ідея: `service` залежить лише від **інтерфейсів** репозиторіїв, які він
сам і оголошує (`StudentRepository`, `CourseRepository`,
`EnrollmentRepository`). Конкретні SQLite-реалізації з пакета `storage`
впроваджуються у `cmd/server` (Dependency Injection). Це дозволяє підмінити
сховище без зміни бізнес-логіки.

## Доменна модель (class diagram)

```mermaid
classDiagram
    class Student {
        +string ID
        +string Name
        +string Email
        +time CreatedAt
        +Validate() error
    }
    class CourseRecord {
        +string ID
        +string Code
        +string Title
        +int Credits
        +CourseType Type
        +string Grading
        +[]string Features
        +string Platform
    }
    class Enrollment {
        +string ID
        +string StudentID
        +string CourseID
        +EnrollmentStatus Status
        +float64 FinalGrade
        +string Letter
        +bool Passed
        +float64 Progress
        +[]Assessment Assessments
    }
    class Assessment {
        +string Name
        +float64 Score
        +float64 MaxScore
        +float64 Weight
    }
    class Course {
        <<interface>>
        +ID() string
        +Code() string
        +Title() string
        +Credits() int
        +GradingStrategy() string
        +Features() []string
        +Describe() string
    }
    Enrollment "1" *-- "many" Assessment
    Student "1" -- "many" Enrollment
    CourseRecord ..> Course : factory builds
    Enrollment ..> Student
    Enrollment ..> Course
```

## Патерни GoF (class diagram)

```mermaid
classDiagram
    %% Strategy
    class Strategy {
        <<interface>>
        +Name() string
        +Evaluate(components) Result
    }
    class WeightedAverage
    class PassFail
    class LetterGrade
    Strategy <|.. WeightedAverage
    Strategy <|.. PassFail
    Strategy <|.. LetterGrade

    %% Factory + Decorator
    class Factory {
        +Build(CourseRecord) Course
    }
    class base
    class CertifiedCourse
    class OnlineCourse
    Course <|.. base
    Course <|.. CertifiedCourse
    Course <|.. OnlineCourse
    CertifiedCourse o-- Course : wraps
    OnlineCourse o-- Course : wraps
    Factory ..> Course : creates

    %% Observer
    class Observer {
        <<interface>>
        +OnGraded(ctx, Event) error
    }
    class Publisher {
        +Subscribe(Observer)
        +Notify(ctx, Event) error
    }
    class ProgressTracker
    class LogNotifier
    Observer <|.. ProgressTracker
    Observer <|.. LogNotifier
    Publisher o-- Observer
```

## Сценарій «Записати оцінку» (sequence diagram)

Тут зустрічаються всі патерни: Strategy обчислює результат, Observer публікує
подію, ProgressTracker персистить прогрес.

```mermaid
sequenceDiagram
    actor Client
    participant HTTP as httpapi.Server
    participant ES as EnrollmentService
    participant Repo as EnrollmentRepository
    participant CS as CourseService (Factory)
    participant Strat as grading.Strategy
    participant Pub as progress.Publisher
    participant Track as ProgressTracker

    Client->>HTTP: PUT /api/enrollments/{id}/grade
    HTTP->>ES: RecordGrade(id, inputs, planned)
    ES->>Repo: GetByID(id)
    Repo-->>ES: Enrollment
    ES->>CS: Get(courseID)
    CS-->>ES: Course (decorated)
    ES->>Strat: Evaluate(components)
    Strat-->>ES: Result{Final, Letter, Passed}
    ES->>Repo: Update(enrollment)
    ES->>Pub: Notify(Event)
    Pub->>Track: OnGraded(Event)
    Track->>Repo: SetProgress(id, fraction)
    ES->>Repo: GetByID(id)
    Repo-->>ES: Enrollment (with progress)
    ES-->>HTTP: Enrollment
    HTTP-->>Client: 200 JSON
```

## Стани зарахування (state diagram)

```mermaid
stateDiagram-v2
    [*] --> pending: Enroll
    pending --> active: RecordGrade (graded < planned)
    active --> active: RecordGrade (graded < planned)
    pending --> completed: RecordGrade (graded >= planned)
    active --> completed: RecordGrade (graded >= planned)
    pending --> dropped: withdraw
    active --> dropped: withdraw
    completed --> [*]
    dropped --> [*]
```
