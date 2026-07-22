import { describe, it, expect, beforeEach } from 'vitest'
import { parseToken, getUserInfo, isAdmin } from './auth'

describe('Auth Utils', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('should parse a valid JWT token', () => {
    // Header: {"alg":"HS256","typ":"JWT"} => eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
    // Payload: {"user_id":1,"username":"testuser","role":"developer","exp":1700000000} => eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6InRlc3R1c2VyIiwicm9sZSI6ImRldmVsb3BlciIsImV4cCI6MTcwMDAwMDAwMH0=
    const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6InRlc3R1c2VyIiwicm9sZSI6ImRldmVsb3BlciIsImV4cCI6MTcwMDAwMDAwMH0=.signature'
    
    const parsed = parseToken(mockToken)
    expect(parsed).not.toBeNull()
    expect(parsed?.username).toBe('testuser')
    expect(parsed?.role).toBe('developer')
  })

  it('should return null for invalid token format', () => {
    expect(parseToken('invalid_token')).toBeNull()
    expect(parseToken('')).toBeNull()
  })

  it('getUserInfo should retrieve and parse token from localStorage', () => {
    const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoyLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiZXhwIjoxNzAwMDAwMDAwfQ==.signature'
    localStorage.setItem('token', mockToken)

    const userInfo = getUserInfo()
    expect(userInfo).not.toBeNull()
    expect(userInfo?.username).toBe('admin')
    expect(userInfo?.role).toBe('admin')
  })

  it('getUserInfo should return null if no token exists', () => {
    expect(getUserInfo()).toBeNull()
  })

  it('isAdmin should return true only for admin role', () => {
    localStorage.setItem('token', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoyLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiZXhwIjoxNzAwMDAwMDAwfQ==.signature')
    expect(isAdmin()).toBe(true)

    localStorage.setItem('token', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6InRlc3R1c2VyIiwicm9sZSI6ImRldmVsb3BlciIsImV4cCI6MTcwMDAwMDAwMH0=.signature')
    expect(isAdmin()).toBe(false)
  })
})
