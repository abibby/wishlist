import { createStore, get, set } from 'idb-keyval'
import { useCallback, useEffect, useState } from 'preact/hooks'

const store = createStore('usePersistentState', 'usePersistentState')

export function usePersistentState<T>(
    name: string,
    defaultValue: T,
): [T, (v: T) => Promise<void>] {
    const [value, setValue] = useState<T>(defaultValue)
    useEffect(() => {
        get(name, store).then(v => {
            if (v === undefined) {
                return
            }

            setValue(v)
        })
    }, [name])

    const update = useCallback(
        async (v: T) => {
            setValue(v)
            await set(name, v, store)
        },
        [name],
    )

    return [value, update]
}
