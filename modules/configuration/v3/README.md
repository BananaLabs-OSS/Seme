# Configuration Contract v3

Configuration v3 (`...4000@...4006`, parent `...4005`) preserves the complete
Configuration v2 bound-plan model. It authenticates Package v4 and Core
Execution v36, allowing defaults, validators, initializers, and argument-source
expressions to reference the v36 execution closure without reinterpreting them
under v35. Runtime values and secrets remain outside the immutable artifact.
