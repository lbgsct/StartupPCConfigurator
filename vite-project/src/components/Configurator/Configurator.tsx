import { useState } from "react";
import CategoryTabs from "../CategoryTabs/CategoryTabs";
import ComponentList from "../ComponentList/ComponentList";
import styles from "./Configurator.module.css";
import { SelectedBuild } from "../SelectedBuild/SelectedBuild";
import { Component } from "../../types/index";

const Configurator = () => {
    const [selectedCategory, setSelectedCategory] = useState<string>("cpu");
    const [selectedComponents, setSelectedComponents] = useState<Record<string, Component | null>>({
        cpu: null,
        gpu: null,
        motherboard: null,
        ram: null,
        hdd: null,
        ssd: null,
        cooler: null,
        case: null,
        psu: null,
    });

    return (
        <div className={styles.container}>
            <CategoryTabs
                onSelect={setSelectedCategory}
                selectedComponents={selectedComponents}
            />

            <ComponentList
                selectedCategory={selectedCategory}
                selectedComponents={selectedComponents}
                setSelectedComponents={setSelectedComponents}
            />

            <SelectedBuild
                selectedComponents={selectedComponents}
                setSelectedComponents={setSelectedComponents}
            />
        </div>
    );
}

export default Configurator;