import { h, RenderableProps } from 'preact'
import { Link } from 'preact-router'
import { useCallback } from 'preact/hooks'
import { useUser } from '../auth'
import { openModal } from '../components/modal'
import { LoginModal } from '../components/modals/login'
import styles from './default.module.css'

h

export function Default({ children }: RenderableProps<never>) {
    const user = useUser()

    const login = useCallback(async () => {
        await openModal(LoginModal, {})
    }, [])

    return (
        <div class={styles.default}>
            <nav class={styles.nav}>
                <Link class={styles.home} href='/'>
                    Wishist
                </Link>
                {user ? (
                    <Link class={styles.logout} href='/account'>
                        Account
                    </Link>
                ) : (
                    <button class={styles.login} onClick={login}>
                        login
                    </button>
                )}
            </nav>
            {children}
        </div>
    )
}
