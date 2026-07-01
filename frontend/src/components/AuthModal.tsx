import { useState } from 'react'
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from './ui/dialog'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { login, register, setAuthToken, setCurrentUser, type LoginRespVO, type UserVO } from '../api'

interface AuthModalProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    onLoginSuccess: () => void
}

export default function AuthModal({ open, onOpenChange, onLoginSuccess }: AuthModalProps) {
    const [mode, setMode] = useState<'login' | 'register'>('login')
    const [username, setUsername] = useState('')
    const [email, setEmail] = useState('')
    const [password, setPassword] = useState('')
    const [confirmPassword, setConfirmPassword] = useState('')
    const [error, setError] = useState('')
    const [loading, setLoading] = useState(false)

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        setError('')

        if (mode === 'register') {
            if (!email) {
                setError('Email is required')
                return
            }
            if (password !== confirmPassword) {
                setError('Passwords do not match')
                return
            }
        }

        setLoading(true)
        try {
            if (mode === 'login') {
                const result: LoginRespVO = await login(username, password)
                setAuthToken(result.token)
                setCurrentUser(result.user)
            } else {
                const user: UserVO = await register(username, email, password)
                // Auto login after register
                const result: LoginRespVO = await login(username, password)
                setAuthToken(result.token)
                setCurrentUser(result.user)
            }
            onOpenChange(false)
            onLoginSuccess()
        } catch (err) {
            setError(err instanceof Error ? err.message : 'An error occurred')
        } finally {
            setLoading(false)
        }
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md !p-0">
                <div style={{ padding: '36px' }}>
                <DialogHeader>
                    <DialogTitle className="text-xl">
                        {mode === 'login' ? 'Sign In' : 'Create Account'}
                    </DialogTitle>
                    <DialogDescription>
                        {mode === 'login'
                            ? 'Enter your credentials to access your account'
                            : 'Create an account to get started'}
                    </DialogDescription>
                </DialogHeader>

                {error && (
                    <div className="mt-4 p-3 bg-red-500/10 text-red-500 rounded-lg text-sm">
                        {error}
                    </div>
                )}

                <form onSubmit={handleSubmit} style={{ marginTop: '24px' }}>
                    <div style={{ marginBottom: '20px' }}>
                        <label className="block text-sm font-medium text-[var(--text)]" style={{ marginBottom: '8px', display: 'block' }}>
                            Username
                        </label>
                        <Input
                            type="text"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            placeholder="Enter your username"
                            required
                            className="w-full"
                        />
                    </div>

                    {mode === 'register' && (
                        <div style={{ marginBottom: '20px' }}>
                            <label className="block text-sm font-medium text-[var(--text)]" style={{ marginBottom: '8px', display: 'block' }}>
                                Email
                            </label>
                            <Input
                                type="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="Enter your email"
                                required
                                className="w-full"
                            />
                        </div>
                    )}

                    <div style={{ marginBottom: '20px' }}>
                        <label className="block text-sm font-medium text-[var(--text)]" style={{ marginBottom: '8px', display: 'block' }}>
                            Password
                        </label>
                        <Input
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            placeholder="Enter your password"
                            required
                            className="w-full"
                        />
                    </div>

                    {mode === 'register' && (
                        <div style={{ marginBottom: '20px' }}>
                            <label className="block text-sm font-medium text-[var(--text)]" style={{ marginBottom: '8px', display: 'block' }}>
                                Confirm Password
                            </label>
                            <Input
                                type="password"
                                value={confirmPassword}
                                onChange={(e) => setConfirmPassword(e.target.value)}
                                placeholder="Confirm your password"
                                required
                                className="w-full"
                            />
                        </div>
                    )}

                    <Button type="submit" disabled={loading} className="w-full" style={{ marginTop: '8px' }}>
                        {loading ? 'Loading...' : mode === 'login' ? 'Sign In' : 'Create Account'}
                    </Button>
                </form>

                <p className="text-center text-sm text-[var(--text-muted)]" style={{ marginTop: '20px' }}>
                    {mode === 'login' ? "Don't have an account?" : 'Already have an account?'}{' '}
                    <button
                        type="button"
                        onClick={() => {
                            setMode(mode === 'login' ? 'register' : 'login')
                            setError('')
                        }}
                        className="text-blue-500 hover:underline"
                    >
                        {mode === 'login' ? 'Sign up' : 'Sign in'}
                    </button>
                </p>
                </div>
            </DialogContent>
        </Dialog>
    )
}