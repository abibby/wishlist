import { Fragment, h } from 'preact'
import { Link } from 'preact-router'
import { FetchError } from '../api/internal'

h

export type ErrorFetchErrorProps = {
    err: FetchError<unknown>
}

type StandardError = {
    error: string
    status: number
}

function isStandardError(v: unknown): v is StandardError {
    return (
        typeof v === 'object' &&
        v !== null &&
        'error' in v &&
        typeof v.error === 'string' &&
        'status' in v &&
        typeof v.status === 'string'
    )
}

export function ErrorFetchError({ err }: ErrorFetchErrorProps) {
    let stdErr: StandardError = {
        error: 'error',
        status: err.status,
    }
    if (isStandardError(err.body)) {
        stdErr = err.body
    }

    return (
        <Fragment>
            <h1>
                {stdErr.status} {stdErr.error}
            </h1>
            <Link href={'/'}>Home</Link>
        </Fragment>
    )
}
