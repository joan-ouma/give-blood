# Blood Donation MVP - API Documentation

The Go backend service (`blood-mvp-api`) exposes a JSON REST API to support the frontend application. It manages donors, agencies, blood drives, and the donation verification flow.

## Base Configuration

* **Base URL**: `http://localhost:8085`
* **API Prefix**: `/api`
* **Content-Type**: `application/json`

---

## Authentication & Authorization

All authenticated endpoints require a JSON Web Token (JWT) passed in the `Authorization` header:

```http
Authorization: Bearer <access_token>
```

### Roles
The system supports two user roles:
* `donor` - Can search drives/locations, log donations, check eligibility, and view the leaderboard.
* `agency` - Can manage locations, manage blood drives, and verify/reject logged donations.

---

## Endpoints Quick Reference

| Endpoint | Method | Role | Description |
| :--- | :--- | :--- | :--- |
| [`/api/auth/register`](#post-apiauthregister) | `POST` | Public | Register a new user (donor or agency) |
| [`/api/auth/login`](#post-apiauthlogin) | `POST` | Public | Authenticate and obtain JWT access token |
| [`/api/auth/refresh`](#post-apiauthrefresh) | `POST` | Public | Refresh expired access token using cookie |
| [`/api/auth/me`](#get-apiauthme) | `GET` | Authenticated | Get current authenticated user details |
| [`/api/locations`](#get-apilocations) | `GET` | Public | List all active blood donation centers |
| [`/api/locations`](#post-apilocations) | `POST` | Agency | Create a new blood donation center |
| [`/api/locations/:id`](#get-apilocationsid) | `GET` | Public | Get details of a specific donation center |
| [`/api/locations/:id`](#put-apilocationsid) | `PUT` | Agency | Update a donation center's details |
| [`/api/locations/:id`](#delete-apilocationsid) | `DELETE` | Agency | Remove a donation center |
| [`/api/drives`](#get-apidrives) | `GET` | Public | List all upcoming blood drives |
| [`/api/drives`](#post-apidrives) | `POST` | Agency | Create an upcoming blood drive |
| [`/api/drives/:id`](#get-apidrivesid) | `GET` | Public | Get details of a specific blood drive |
| [`/api/drives/:id`](#put-apidrivesid) | `PUT` | Agency | Update blood drive details |
| [`/api/drives/:id`](#delete-apidrivesid) | `DELETE` | Agency | Cancel/remove a blood drive |
| [`/api/donations`](#post-apidonations) | `POST` | Donor | Log a new blood donation |
| [`/api/donations/mine`](#get-apidonationsmine) | `GET` | Donor | Get logged-in donor's donation logs |
| [`/api/agency/donations/pending`](#get-apiagencydonationspending) | `GET` | Agency | Get pending donations awaiting verification |
| [`/api/donations/:id/verify`](#post-apidonationsidverify) | `POST` | Agency | Approve/verify a donation |
| [`/api/donations/:id/reject`](#post-apidonationsidreject) | `POST` | Agency | Reject a donation log with a reason |
| [`/api/donors/me/eligibility`](#get-apidonorsmeeligibility) | `GET` | Donor | Check donation eligibility/cooldown status |
| [`/api/leaderboard`](#get-apileaderboard) | `GET` | Public | Get donor points leaderboard |

---

## Endpoint Details

### Authentication

#### `POST /api/auth/register`
Create a new user account.

**Request Body (`RegisterRequest`)**:
```json
{
  "email": "donor@example.com",
  "password": "securepassword123",
  "role": "donor",
  "name": "Jane Doe"
}
```
* **Validation**:
  * `email` is required.
  * `password` must be at least 8 characters.
  * `role` must be either `donor` or `agency`.
  * `name` is required.

**Responses**:
* **201 Created**: Returns the created user object.
* **400 Bad Request**: Validation errors or email already registered.
  ```json
  {
    "error": "validation failed",
    "fields": {
      "password": "Password must be at least 8 characters"
    }
  }
  ```

---

#### `POST /api/auth/login`
Authenticate credentials and establish session.

**Request Body (`LoginRequest`)**:
```json
{
  "email": "donor@example.com",
  "password": "securepassword123"
}
```

**Responses**:
* **200 OK**: Sets an HttpOnly `refreshToken` cookie (valid for 7 days) and returns the short-lived access token:
  ```json
  {
    "accessToken": "eyJhbGciOiJIUzI1Ni..."
  }
  ```
* **400 Bad Request**: Invalid email or password.
* **429 Too Many Requests**: Rate-limited due to too many failed attempts.

---

#### `POST /api/auth/refresh`
Refresh the short-lived access token. Must send the HttpOnly `refreshToken` cookie.

**Responses**:
* **200 OK**:
  ```json
  {
    "accessToken": "eyJhbGciOiJIUzI1Ni..."
  }
  ```
* **401 Unauthorized**: Missing, expired, or invalid refresh token.

---

#### `GET /api/auth/me`
Retrieve user profile details for the authenticated session.

**Responses**:
* **200 OK**:
  ```json
  {
    "id": "64efc14a2df84a0d9c...",
    "email": "donor@example.com",
    "role": "donor",
    "name": "Jane Doe"
  }
  ```
* **401 Unauthorized**: Missing or invalid access token.

---

### Locations (Donation Centers)

#### `GET /api/locations`
List all active donation centers.

**Query Parameters**:
* `city` (optional): Filter locations by city name.

**Responses**:
* **200 OK**: Array of `LocationResponse` objects.

---

#### `POST /api/locations`
Create a new donation center. (Required Role: `agency`)

**Request Body (`LocationCreateRequest`)**:
```json
{
  "name": "Central Blood Bank",
  "address": "123 Health St",
  "city": "Metropolis",
  "hours": "Mon-Fri 8:00 AM - 5:00 PM",
  "phone": "+15550199",
  "lat": 40.7128,
  "lng": -74.0060
}
```

**Responses**:
* **201 Created**: Returns the created `LocationResponse`.
* **403 Forbidden**: If the user is not an agency.

---

#### `GET /api/locations/:id`
Get a specific donation center.

**Responses**:
* **200 OK**: `LocationResponse` object.
* **404 Not Found**: Center does not exist.

---

#### `PUT /api/locations/:id`
Update center details. (Required Role: `agency` & Creator of center)

**Request Body (`LocationUpdateRequest`)**: Same fields as `LocationCreateRequest`.

---

#### `DELETE /api/locations/:id`
Delete a donation center. (Required Role: `agency` & Creator of center)

**Responses**:
* **200 OK**:
  ```json
  {
    "success": true
  }
  ```

---

### Blood Drives

#### `GET /api/drives`
List all blood drives.

**Query Parameters**:
* `city` (optional): Filter drives by city name.

**Responses**:
* **200 OK**: Array of `DriveResponse` objects.

---

#### `POST /api/drives`
Create a new blood drive. (Required Role: `agency`)

**Request Body (`DriveCreateRequest`)**:
```json
{
  "locationId": "64efc14a2df84a0d9c...",
  "title": "Summer Blood Drive 2026",
  "city": "Metropolis",
  "startsAt": "2026-08-20T09:00:00Z",
  "endsAt": "2026-08-20T17:00:00Z",
  "notes": "Free snacks and t-shirts for all donors!"
}
```

**Responses**:
* **201 Created**: Returns `DriveResponse`.

---

### Donation Logs

#### `POST /api/donations`
Log a new donation from a blood drive or donation center. (Required Role: `donor`)

**Request Body (`DonationCreateRequest`)**:
```json
{
  "locationId": "64efc14a2df84a0d9c...",
  "pints": 1,
  "donatedAt": "2026-08-16T12:00:00Z"
}
```
* **Validation**:
  * Either `driveId` or `locationId` must be provided.
  * `pints` must be between `1` and `2`.
  * `donatedAt` must be a valid RFC3339 timestamp in the past.

**Responses**:
* **201 Created**: Returns `DonationResponse` (initial status will be `pending`).
* **429 Too Many Requests**: Logged-in donor is rate-limited.

---

#### `GET /api/donations/mine`
Get the logged-in donor's historical logs. (Required Role: `donor`)

**Query Parameters**:
* `limit` (optional): Default `20`, maximum `100`.
* `offset` (optional): Default `0`.

**Responses**:
* **200 OK**: Array of `DonationResponse` sorted by `donatedAt` descending.

---

#### `GET /api/agency/donations/pending`
Get pending donation logs submitted for drives/locations owned by this agency. (Required Role: `agency`)

**Responses**:
* **200 OK**: Array of pending `DonationResponse` logs.

---

#### `POST /api/donations/:id/verify`
Verify and approve a pending donation. (Required Role: `agency` & Owner of the drive/location)

**Responses**:
* **200 OK**: Returns updated `DonationResponse` with status `verified`.
* **409 Conflict**: If the donation log is already verified or rejected.

---

#### `POST /api/donations/:id/reject`
Reject a pending donation log with a reason. (Required Role: `agency` & Owner of the drive/location)

**Request Body (`DonationRejectRequest`)**:
```json
{
  "rejectionReason": "Name mismatch on donor record."
}
```

**Responses**:
* **200 OK**: Returns updated `DonationResponse` with status `rejected`.
* **409 Conflict**: If the donation log is already processed.

---

### Eligibility & Leaderboards

#### `GET /api/donors/me/eligibility`
Check the logged-in donor's eligibility status. Calculates the 56-day cooldown since the last verified donation. (Required Role: `donor`)

**Responses**:
* **200 OK**:
  ```json
  {
    "lastDonationAt": "2026-06-20T12:00:00Z",
    "nextEligibleAt": "2026-08-15T12:00:00Z",
    "isEligibleNow": true,
    "daysRemaining": 0
  }
  ```

---

#### `GET /api/leaderboard`
Retrieve the global points leaderboard of donors.

Points are calculated as:
`Points = (Total Verified Donations * 10) + (Total Pints * 5)`

**Responses**:
* **200 OK**:
  ```json
  {
    "entries": [
      {
        "rank": 1,
        "donorId": "64efc14a2df84a0d9c...",
        "name": "Jane Doe",
        "points": 45,
        "totalDonations": 3,
        "totalPints": 3
      }
    ],
    "me": {
      "rank": 1,
      "donorId": "64efc14a2df84a0d9c...",
      "name": "Jane Doe",
      "points": 45,
      "totalDonations": 3,
      "totalPints": 3
    }
  }
  ```
  *(Note: `me` contains the logged-in user's entry even if they are not in the top visible rank range).*
