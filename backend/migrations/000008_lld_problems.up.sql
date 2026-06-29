-- 000008_lld_problems.up.sql
-- Low-Level Design catalog per docs/04-DATABASE-SCHEMA.md §4.2 (lld_problems).
--
-- Content table: migration-managed and seeded. Carries deleted_at so a problem
-- can be retired without breaking FK history. Sections are stored as
-- TEXT/markdown plus JSONB lists (design_patterns, follow_up_questions).
--
-- Seeds 7 classic LLD interview problems idempotently via
-- INSERT ... ON CONFLICT (slug) DO NOTHING so the migration is safe to re-run
-- and never duplicates rows.

BEGIN;

-- lld_problems --------------------------------------------------------------
CREATE TABLE IF NOT EXISTS lld_problems (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    track_id            UUID NOT NULL REFERENCES tracks (id) ON DELETE RESTRICT,
    pillar_id           UUID NULL REFERENCES pillars (id) ON DELETE SET NULL,
    slug                TEXT NOT NULL,
    title               TEXT NOT NULL,
    difficulty          difficulty NOT NULL,
    order_index         INTEGER NOT NULL DEFAULT 0,
    requirements_md     TEXT NULL,
    entities_md         TEXT NULL,
    class_diagram_md    TEXT NULL,
    design_patterns     JSONB NOT NULL DEFAULT '[]',
    solid_notes_md      TEXT NULL,
    api_or_interface_md TEXT NULL,
    tradeoffs_md        TEXT NULL,
    follow_up_questions JSONB NOT NULL DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_lld_problems_slug ON lld_problems (slug);
CREATE INDEX IF NOT EXISTS idx_lld_order ON lld_problems (track_id, order_index);

DROP TRIGGER IF EXISTS trg_lld_problems_updated_at ON lld_problems;
CREATE TRIGGER trg_lld_problems_updated_at BEFORE UPDATE ON lld_problems
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Ensure the backend-sde3 track and lld pillar exist so the seed FKs resolve
-- even when this migration runs before the Go content seeder. Both upserts are
-- idempotent; the Go seeder later reconciles names/descriptions.
INSERT INTO tracks (slug, name, description, seniority, is_active, sort_order)
VALUES ('backend-sde3', 'Backend SDE3',
        'Senior backend engineering interview preparation track.', 'SDE3', true, 0)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO pillars (track_id, type, name, description, weight, sort_order)
SELECT t.id, 'lld', 'Low-Level Design',
       'Object-oriented design and design patterns.', 1.0, 2
FROM tracks t
WHERE t.slug = 'backend-sde3'
ON CONFLICT (track_id, type) DO NOTHING;

-- Seed: 7 classic LLD problems -----------------------------------------------
-- Each row resolves track_id/pillar_id by slug/type so it is independent of
-- generated UUIDs. ON CONFLICT (slug) DO NOTHING keeps the seed idempotent.
INSERT INTO lld_problems (
    track_id, pillar_id, slug, title, difficulty, order_index,
    requirements_md, entities_md, class_diagram_md, design_patterns,
    solid_notes_md, api_or_interface_md, tradeoffs_md, follow_up_questions
)
SELECT
    t.id,
    p.id,
    v.slug, v.title, v.difficulty::difficulty, v.order_index,
    v.requirements_md, v.entities_md, v.class_diagram_md, v.design_patterns::jsonb,
    v.solid_notes_md, v.api_or_interface_md, v.tradeoffs_md, v.follow_up_questions::jsonb
FROM (
    VALUES
    (
        'parking-lot', 'Design a Parking Lot', 'medium', 1,
        'Functional: park and unpark vehicles of different sizes (motorcycle, car, bus); assign the nearest available compatible spot; issue a ticket on entry and compute a fee on exit based on duration and vehicle/spot type; support multiple floors and entry/exit gates. Non-functional: support concurrent gate operations safely, be extensible to new vehicle types and pricing strategies, and keep spot lookup close to O(1).',
        'Core entities: ParkingLot (aggregate of ParkingFloor), ParkingFloor (rows of ParkingSpot), ParkingSpot (with a SpotType: Compact/Large/Motorcycle/Handicapped/Electric), Vehicle (abstract; Car/Bus/Motorcycle subclasses), Ticket (entry time, spot, vehicle), EntryGate/ExitGate, ParkingAttendant, and a FeeCalculator/PricingStrategy.',
        '```mermaid
classDiagram
  ParkingLot "1" o-- "*" ParkingFloor
  ParkingFloor "1" o-- "*" ParkingSpot
  Vehicle <|-- Car
  Vehicle <|-- Bus
  Vehicle <|-- Motorcycle
  ParkingSpot --> Vehicle
  Ticket --> ParkingSpot
  Ticket --> Vehicle
  ParkingLot --> FeeCalculator
```',
        '["Strategy (pricing/fee calculation)", "Factory (vehicle and spot creation)", "Singleton (single ParkingLot instance)", "Observer (display board availability updates)"]',
        'Single Responsibility: separate FeeCalculator from ParkingLot allocation. Open/Closed: add new PricingStrategy or Vehicle subtypes without editing existing allocation code. Liskov: every Vehicle subtype is parkable where its size allows. Dependency Inversion: ParkingLot depends on a PricingStrategy interface, not a concrete calculator.',
        'park(vehicle) -> Ticket; unpark(ticket) -> Fee; getAvailableSpot(vehicleType) -> ParkingSpot; FeeCalculator.calculate(ticket, exitTime) -> Money.',
        'Per-spot locking vs per-floor locking trades throughput against complexity. Pre-computing nearest-spot indexes speeds allocation but costs memory; a simple per-type free-list keeps it O(1) at the cost of strict nearest-spot ordering.',
        '["How would you support electric-charging spots with reservation?", "How do you make spot allocation thread-safe across gates?", "How would pricing change for monthly pass holders?"]'
    ),
    (
        'movie-ticket-booking', 'Design BookMyShow (Movie Ticket Booking)', 'hard', 2,
        'Functional: browse cities, cinemas, movies and shows; view a seat map for a show; select seats and book them; process payment; prevent double-booking of the same seat. Non-functional: strong consistency on seat reservation under high concurrency, temporary seat holds with expiry, and horizontal scalability across cities.',
        'Core entities: City, Cinema, Hall/Screen, Show (movie + hall + time), Seat and ShowSeat (status: Available/Held/Booked), Movie, Booking, Payment, User, and a SeatLockManager/ReservationService.',
        '```mermaid
classDiagram
  City "1" o-- "*" Cinema
  Cinema "1" o-- "*" Hall
  Hall "1" o-- "*" Seat
  Show --> Hall
  Show --> Movie
  Show "1" o-- "*" ShowSeat
  Booking --> ShowSeat
  Booking --> Payment
  Booking --> User
```',
        '["State (ShowSeat lifecycle: available -> held -> booked)", "Strategy (payment methods)", "Observer (notify on booking confirmation)", "Factory (payment processor creation)", "Singleton (SeatLockManager)"]',
        'Single Responsibility: ReservationService owns locking; PaymentService owns payments. Open/Closed: add payment methods via new Strategy implementations. Dependency Inversion: Booking depends on a PaymentGateway interface. Interface Segregation: separate read (seat map) and write (reserve) interfaces.',
        'searchShows(city, movie) -> [Show]; getSeatMap(showId) -> [ShowSeat]; holdSeats(showId, seatIds, userId) -> HoldToken; confirmBooking(holdToken, payment) -> Booking.',
        'Pessimistic seat locks give correctness but reduce concurrency and need expiry handling; optimistic locking scales better but forces retries. A distributed lock (e.g. Redis) adds availability risk vs a DB row lock that is simpler but couples to one DB.',
        '["How do you expire seat holds that were never paid for?", "How do you prevent two users booking the same seat simultaneously?", "How would you scale the seat map for a blockbuster opening?"]'
    ),
    (
        'splitwise', 'Design Splitwise (Expense Sharing)', 'medium', 3,
        'Functional: create users and groups; add an expense split equally, by exact amount, or by percentage; track per-user balances (who owes whom); simplify debts to minimize transactions; settle up. Non-functional: accurate money arithmetic, auditable expense history, and extensible split strategies.',
        'Core entities: User, Group, Expense (payer, amount, participants), Split (abstract; EqualSplit/ExactSplit/PercentSplit), Balance/BalanceSheet, Transaction/Settlement, and an ExpenseManager plus a DebtSimplifier.',
        '```mermaid
classDiagram
  Group "1" o-- "*" User
  Group "1" o-- "*" Expense
  Expense "1" o-- "*" Split
  Split <|-- EqualSplit
  Split <|-- ExactSplit
  Split <|-- PercentSplit
  User "1" o-- "*" Balance
  ExpenseManager --> DebtSimplifier
```',
        '["Strategy (split algorithms)", "Factory (split creation by type)", "Observer (balance update notifications)", "Singleton (ExpenseManager)"]',
        'Single Responsibility: Split validation separate from balance updates. Open/Closed: add a new split type by adding a Split subclass. Liskov: every Split subtype produces shares summing to the expense total. Dependency Inversion: ExpenseManager depends on the Split abstraction.',
        'addExpense(payer, amount, splitType, shares) -> Expense; getBalances(userId) -> [Balance]; simplifyDebts(groupId) -> [Transaction]; settleUp(fromUser, toUser, amount) -> Settlement.',
        'Storing running balances gives fast reads but risks drift; recomputing from expenses is always correct but slower. Debt simplification reduces transaction count but can obscure the original who-paid-what trail, so keep the raw expense log for audit.',
        '["How do you keep money math exact (avoid floating-point errors)?", "How would you simplify debts across a large group?", "How do you handle multi-currency expenses?"]'
    ),
    (
        'elevator-system', 'Design an Elevator System', 'medium', 4,
        'Functional: handle internal car requests (floor buttons) and external hall calls (up/down); dispatch the most suitable car; move cars and open/close doors; report current floor and direction. Non-functional: minimize average wait time, be safe (no overload, no conflicting moves), and support a configurable scheduling policy for a bank of elevators.',
        'Core entities: ElevatorSystem (the controller for a bank), ElevatorCar (current floor, direction, door state, request queue), Request (internal/external), Button (HallButton/CarButton), Door, Display, and a SchedulingStrategy/Dispatcher.',
        '```mermaid
classDiagram
  ElevatorSystem "1" o-- "*" ElevatorCar
  ElevatorSystem --> Dispatcher
  ElevatorCar --> Door
  ElevatorCar --> Display
  ElevatorCar "1" o-- "*" Request
  Dispatcher --> SchedulingStrategy
```',
        '["Strategy (scheduling/dispatch policy)", "State (car state: moving up/down, idle, doors open)", "Observer (button presses notify the controller)", "Command (encapsulate requests)", "Singleton (ElevatorSystem controller)"]',
        'Single Responsibility: dispatching logic lives in the Dispatcher, not the car. Open/Closed: swap scheduling policies (FCFS, SCAN/LOOK, nearest-car) without changing the car. State pattern keeps car-state transitions explicit and safe. Dependency Inversion: Dispatcher depends on a SchedulingStrategy interface.',
        'requestElevator(floor, direction); selectFloor(carId, floor); step() to advance simulation; Dispatcher.assign(request) -> ElevatorCar.',
        'A simple FCFS queue is easy but yields poor wait times; SCAN/LOOK improves throughput but starves opposite-direction calls without tuning. Centralized dispatch is simpler to reason about; per-car autonomy scales but risks suboptimal global assignment.',
        '["Which scheduling algorithm minimizes average wait time?", "How do you model door-open/close timing safely?", "How would you extend this to a multi-building elevator bank?"]'
    ),
    (
        'chess-game', 'Design a Chess Game', 'hard', 5,
        'Functional: represent an 8x8 board and all pieces; generate and validate legal moves per piece including special moves (castling, en passant, promotion); detect check, checkmate, and stalemate; alternate turns between two players; maintain move history. Non-functional: correct rule enforcement, extensibility to variants, and clean separation of rules from board state.',
        'Core entities: Game (orchestrates play), Board (8x8 grid of Cell), Piece (abstract; King/Queen/Rook/Bishop/Knight/Pawn each with movement rules), Cell/Position, Player, Move, and a MoveValidator plus a GameStatus.',
        '```mermaid
classDiagram
  Game --> Board
  Game "1" o-- "2" Player
  Board "1" o-- "*" Cell
  Cell --> Piece
  Piece <|-- King
  Piece <|-- Queen
  Piece <|-- Rook
  Piece <|-- Bishop
  Piece <|-- Knight
  Piece <|-- Pawn
  Game "1" o-- "*" Move
```',
        '["Strategy (per-piece movement rules)", "Factory (piece creation / board setup)", "State (game status: active, check, checkmate, stalemate)", "Command (moves for undo/redo and history)", "Memento (board snapshots for undo)"]',
        'Single Responsibility: each Piece owns only its own movement generation; MoveValidator owns legality (check exposure). Open/Closed: add a new piece or variant by subclassing Piece. Liskov: every Piece honors the canMove contract. Dependency Inversion: Game depends on the Piece abstraction, not concrete types.',
        'move(from, to) -> MoveResult; getLegalMoves(position) -> [Move]; isInCheck(color) -> bool; getStatus() -> GameStatus.',
        'Generating all legal moves up front is simple but expensive; lazy validation per attempted move is cheaper but spreads rules around. Storing full board snapshots makes undo trivial at a memory cost, versus reversible Command moves that are compact but trickier for special moves.',
        '["How do you detect checkmate vs stalemate?", "How would you implement undo/redo?", "How would you support chess variants like Chess960?"]'
    ),
    (
        'food-delivery', 'Design a Food Delivery System (Swiggy/DoorDash)', 'hard', 6,
        'Functional: customers browse nearby restaurants and menus, place an order, and pay; restaurants accept and prepare orders; delivery partners are assigned and tracked; order status updates flow to the customer. Non-functional: low-latency restaurant discovery by location, reliable order state transitions, and scalable partner assignment.',
        'Core entities: User/Customer, Restaurant, MenuItem, Cart, Order (with OrderStatus), Payment, DeliveryPartner, Address/Location, and managers: OrderManager, AssignmentService (partner matching), and NotificationService.',
        '```mermaid
classDiagram
  Customer --> Cart
  Cart "1" o-- "*" MenuItem
  Restaurant "1" o-- "*" MenuItem
  Order --> Restaurant
  Order --> Customer
  Order --> Payment
  Order --> DeliveryPartner
  OrderManager --> AssignmentService
```',
        '["State (order lifecycle: placed -> accepted -> preparing -> out-for-delivery -> delivered)", "Strategy (partner assignment and pricing/surge)", "Observer (status update notifications)", "Factory (payment processors)", "Singleton (OrderManager)"]',
        'Single Responsibility: AssignmentService matches partners; OrderManager owns the order state machine. Open/Closed: add assignment strategies (nearest, least-busy, batched) without touching order code. Dependency Inversion: OrderManager depends on AssignmentService and PaymentGateway interfaces.',
        'searchRestaurants(location) -> [Restaurant]; placeOrder(cart, payment) -> Order; assignPartner(orderId) -> DeliveryPartner; updateStatus(orderId, status); trackOrder(orderId) -> Location.',
        'Push-based partner assignment is fast but can overload nearby partners; auction/pull-based assignment balances load but adds latency. A strict state machine prevents invalid transitions at the cost of flexibility for edge cases like reassignment after a partner cancels.',
        '["How do you assign the best delivery partner in real time?", "How do you model and enforce valid order state transitions?", "How would you handle surge pricing?"]'
    ),
    (
        'ride-sharing', 'Design a Ride Sharing System (Uber/Lyft)', 'hard', 7,
        'Functional: riders request a ride from pickup to drop-off; match the nearest available driver; compute fare with optional surge; track the trip; complete payment and collect ratings. Non-functional: low-latency driver matching by geolocation, accurate fare computation, and reliable trip state management at scale.',
        'Core entities: User (Rider/Driver), Vehicle, Location, Trip (with TripStatus), Fare/PricingStrategy, MatchingService (driver-rider matching), Payment, Rating, and a TripManager.',
        '```mermaid
classDiagram
  User <|-- Rider
  User <|-- Driver
  Driver --> Vehicle
  Trip --> Rider
  Trip --> Driver
  Trip --> Fare
  Trip --> Payment
  TripManager --> MatchingService
  MatchingService --> Location
```',
        '["Strategy (matching algorithm and surge pricing)", "State (trip lifecycle: requested -> matched -> in-progress -> completed)", "Observer (trip and location update notifications)", "Factory (payment processor creation)", "Singleton (TripManager)"]',
        'Single Responsibility: MatchingService matches; PricingStrategy computes fare; TripManager owns trip state. Open/Closed: swap matching or pricing strategies without editing trip code. Dependency Inversion: TripManager depends on MatchingService and PricingStrategy interfaces.',
        'requestRide(rider, pickup, dropoff) -> Trip; matchDriver(tripId) -> Driver; computeFare(trip) -> Money; startTrip(tripId); endTrip(tripId) -> Payment.',
        'A geospatial index (quadtree/geohash/H3) speeds nearest-driver queries but adds maintenance overhead versus a naive linear scan that does not scale. Surge pricing improves supply-demand balance but complicates fare predictability and must be applied consistently across retries.',
        '["How do you efficiently find the nearest available driver?", "How do you compute surge pricing fairly?", "How do you keep trip state consistent if a driver disconnects mid-trip?"]'
    )
) AS v (
    slug, title, difficulty, order_index,
    requirements_md, entities_md, class_diagram_md, design_patterns,
    solid_notes_md, api_or_interface_md, tradeoffs_md, follow_up_questions
)
CROSS JOIN tracks t
LEFT JOIN pillars p ON p.track_id = t.id AND p.type = 'lld'
WHERE t.slug = 'backend-sde3'
ON CONFLICT (slug) DO NOTHING;

COMMIT;
